package plaid

import (
	"context"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type Writer struct {
	tx bob.Tx
	Reader
}

func NewWriter(tx bob.Tx) *Writer {
	return &Writer{
		tx:     tx,
		Reader: Reader{exec: tx},
	}
}

func (w *Writer) CreateItem(ctx context.Context, create *PlaidItemCreate) (uuid.UUID, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, err
	}
	setter := &bobgen.PlaidItemSetter{
		ID:              omit.From(id),
		AccessToken:     omit.From(create.AccessToken),
		PlaidItemID:     omit.From(create.PlaidItemID),
		InstitutionID:   omit.From(create.InstitutionID),
		InstitutionName: omit.From(create.InstitutionName),
	}
	row, err := bobgen.PlaidItems.Insert(setter).One(ctx, w.tx)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

func (w *Writer) UpdateCursor(ctx context.Context, itemID uuid.UUID, cursor string) error {
	setter := bobgen.PlaidItemSetter{
		Cursor: omit.From(cursor),
	}
	_, err := bobgen.PlaidItems.Update(
		setter.UpdateMod(),
		um.Where(bobgen.PlaidItems.Columns.ID.EQ(psql.Arg(itemID))),
	).Exec(ctx, w.tx)
	return err
}

func (w *Writer) CreateAccountLink(ctx context.Context, link *AccountLink) error {
	setter := &bobgen.PlaidAccountLinkSetter{
		PlaidAccountID: omit.From(link.PlaidAccountID),
		AccountID:      omit.From(link.AccountID),
		PlaidItemID:    omit.From(link.PlaidItemID),
	}
	_, err := bobgen.PlaidAccountLinks.Insert(setter).One(ctx, w.tx)
	return err
}

func (w *Writer) CreateTransactionLink(ctx context.Context, link *TransactionLink) error {
	setter := &bobgen.PlaidTransactionLinkSetter{
		PlaidTransactionID: omit.From(link.PlaidTransactionID),
		TransactionID:      omit.From(link.TransactionID),
		PlaidAccountID:     omit.From(link.PlaidAccountID),
	}
	_, err := bobgen.PlaidTransactionLinks.Insert(setter).One(ctx, w.tx)
	return err
}

func (w *Writer) DeleteTransactionLink(ctx context.Context, plaidTxnID string) error {
	_, err := bobgen.PlaidTransactionLinks.Delete(
		bobgen.DeleteWhere.PlaidTransactionLinks.PlaidTransactionID.EQ(plaidTxnID),
	).Exec(ctx, w.tx)
	return err
}
