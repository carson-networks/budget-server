package plaid

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/types/pgtypes"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

func (r *Reader) FindItemByID(ctx context.Context, id uuid.UUID) (*PlaidItem, error) {
	row, err := bobgen.FindPlaidItem(ctx, r.exec, id)
	if err != nil {
		return nil, err
	}
	return bobPlaidItemToItem(row), nil
}

func (r *Reader) FindItemByPlaidID(ctx context.Context, plaidItemID string) (*PlaidItem, error) {
	row, err := bobgen.PlaidItems.Query(
		bobgen.SelectWhere.PlaidItems.PlaidItemID.EQ(plaidItemID),
	).One(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return bobPlaidItemToItem(row), nil
}

func (r *Reader) ListItems(ctx context.Context) ([]*PlaidItem, error) {
	rows, err := bobgen.PlaidItems.Query(
		sm.OrderBy(bobgen.PlaidItems.Columns.CreatedAt).Asc(),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertItems(rows), nil
}

// ListItemsWithLinks returns all PlaidItems that have at least one account linked.
func (r *Reader) ListItemsWithLinks(ctx context.Context) ([]*PlaidItem, error) {
	return r.ListItemsForSync(ctx, nil)
}

// ListItemsForSync returns the PlaidItems to sync. If accountIDs is empty all items with at
// least one linked account are returned. Otherwise only items linked to the given accounts
// are returned. Either way the result is deduplicated and ordered by created_at.
func (r *Reader) ListItemsForSync(ctx context.Context, accountIDs []uuid.UUID) ([]*PlaidItem, error) {
	var links bobgen.PlaidAccountLinkSlice
	var err error

	if len(accountIDs) == 0 {
		links, err = bobgen.PlaidAccountLinks.Query().All(ctx, r.exec)
	} else {
		ids := pgtypes.Array[uuid.UUID](accountIDs)
		inExpr := psql.Select(sm.Columns(psql.F("unnest", psql.Cast(psql.Arg(ids), "uuid[]"))))
		links, err = bobgen.PlaidAccountLinks.Query(
			sm.Where(psql.Group(bobgen.PlaidAccountLinks.Columns.AccountID).OP("IN", inExpr)),
		).All(ctx, r.exec)
	}
	if err != nil {
		return nil, err
	}
	if len(links) == 0 {
		return nil, nil
	}

	seen := make(map[uuid.UUID]bool, len(links))
	itemIDs := make(pgtypes.Array[uuid.UUID], 0, len(links))
	for _, l := range links {
		if !seen[l.PlaidItemID] {
			seen[l.PlaidItemID] = true
			itemIDs = append(itemIDs, l.PlaidItemID)
		}
	}

	inExpr := psql.Select(sm.Columns(psql.F("unnest", psql.Cast(psql.Arg(itemIDs), "uuid[]"))))
	rows, err := bobgen.PlaidItems.Query(
		sm.Where(psql.Group(bobgen.PlaidItems.Columns.ID).OP("IN", inExpr)),
		sm.OrderBy(bobgen.PlaidItems.Columns.CreatedAt).Asc(),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertItems(rows), nil
}

// ListAccountLinksByAccountIDs returns AccountLinks for the given internal account UUIDs.
func (r *Reader) ListAccountLinksByAccountIDs(ctx context.Context, accountIDs []uuid.UUID) ([]*AccountLink, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	ids := pgtypes.Array[uuid.UUID](accountIDs)
	inExpr := psql.Select(sm.Columns(psql.F("unnest", psql.Cast(psql.Arg(ids), "uuid[]"))))
	rows, err := bobgen.PlaidAccountLinks.Query(
		sm.Where(psql.Group(bobgen.PlaidAccountLinks.Columns.AccountID).OP("IN", inExpr)),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertAccountLinks(rows), nil
}

// ListAccountLinksByItemID returns all AccountLinks for a given PlaidItem.
func (r *Reader) ListAccountLinksByItemID(ctx context.Context, itemID uuid.UUID) ([]*AccountLink, error) {
	rows, err := bobgen.PlaidAccountLinks.Query(
		bobgen.SelectWhere.PlaidAccountLinks.PlaidItemID.EQ(itemID),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertAccountLinks(rows), nil
}

// FindTransactionLink returns the TransactionLink for a given Plaid transaction ID, or nil if not found.
func (r *Reader) FindTransactionLink(ctx context.Context, plaidTxnID string) (*TransactionLink, error) {
	row, err := bobgen.FindPlaidTransactionLink(ctx, r.exec, plaidTxnID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return bobTransactionLinkToLink(row), nil
}

// -- conversion helpers --

func bobPlaidItemToItem(row *bobgen.PlaidItem) *PlaidItem {
	return &PlaidItem{
		ID:              row.ID,
		AccessToken:     row.AccessToken,
		PlaidItemID:     row.PlaidItemID,
		InstitutionID:   row.InstitutionID,
		InstitutionName: row.InstitutionName,
		Cursor:          row.Cursor,
		CreatedAt:       row.CreatedAt,
	}
}

func convertItems(rows bobgen.PlaidItemSlice) []*PlaidItem {
	out := make([]*PlaidItem, len(rows))
	for i, r := range rows {
		out[i] = bobPlaidItemToItem(r)
	}
	return out
}

func bobAccountLinkToLink(row *bobgen.PlaidAccountLink) *AccountLink {
	return &AccountLink{
		PlaidAccountID: row.PlaidAccountID,
		AccountID:      row.AccountID,
		PlaidItemID:    row.PlaidItemID,
	}
}

func convertAccountLinks(rows bobgen.PlaidAccountLinkSlice) []*AccountLink {
	out := make([]*AccountLink, len(rows))
	for i, r := range rows {
		out[i] = bobAccountLinkToLink(r)
	}
	return out
}

func bobTransactionLinkToLink(row *bobgen.PlaidTransactionLink) *TransactionLink {
	return &TransactionLink{
		PlaidTransactionID: row.PlaidTransactionID,
		TransactionID:      row.TransactionID,
		PlaidAccountID:     row.PlaidAccountID,
	}
}
