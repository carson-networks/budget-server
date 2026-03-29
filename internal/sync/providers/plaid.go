package providers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	budgetsync "github.com/carson-networks/budget-server/internal/sync"
)

type plaidClient interface {
	SyncTransactions(ctx context.Context, accessToken, cursor string) (*plaidclient.SyncResult, error)
	GetAccountBalances(ctx context.Context, accessToken string) ([]plaidclient.AccountBalance, error)
}

type PlaidProvider struct {
	client plaidClient
}

func NewPlaidProvider(client plaidClient) *PlaidProvider {
	return &PlaidProvider{client: client}
}

func (p *PlaidProvider) Type() syncstore.SyncType {
	return syncstore.SyncType_Plaid
}

func (p *PlaidProvider) Sync(ctx context.Context, reader *storage.Reader, accountIDs []uuid.UUID) (*budgetsync.SyncResult, error) {
	items, err := reader.Plaid.ListItemsForSync(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("listing plaid items: %w", err)
	}
	if len(items) == 0 {
		return &budgetsync.SyncResult{}, nil
	}

	byAccount := make(map[uuid.UUID][]actions.IAction)
	var onSuccess []actions.IAction

	for _, item := range items {
		if err := p.syncItem(ctx, reader, item, byAccount, &onSuccess); err != nil {
			return nil, err
		}
	}

	return &budgetsync.SyncResult{ByAccount: byAccount, OnSuccess: onSuccess}, nil
}

func (p *PlaidProvider) syncItem(ctx context.Context, reader *storage.Reader, item *plaidstore.PlaidItem, byAccount map[uuid.UUID][]actions.IAction, onSuccess *[]actions.IAction) error {
	links, err := reader.Plaid.ListAccountLinksByItemID(ctx, item.ID)
	if err != nil {
		return fmt.Errorf("listing account links for item %s: %w", item.ID, err)
	}
	plaidToInternal := make(map[string]uuid.UUID, len(links))
	for _, link := range links {
		plaidToInternal[link.PlaidAccountID] = link.AccountID
	}

	added, modified, removed, nextCursor, err := p.fetchAllPages(ctx, item)
	if err != nil {
		return fmt.Errorf("fetching plaid sync pages for item %s: %w", item.ID, err)
	}

	for _, t := range added {
		internalAccID, ok := plaidToInternal[t.PlaidAccountID]
		if !ok {
			continue
		}
		action, err := p.buildAddAction(t, internalAccID)
		if err != nil {
			return err
		}
		byAccount[internalAccID] = append(byAccount[internalAccID], action)
	}

	for _, t := range modified {
		internalAccID, ok := plaidToInternal[t.PlaidAccountID]
		if !ok {
			continue
		}
		action, err := p.buildModifyAction(ctx, reader, t, internalAccID)
		if err != nil {
			return err
		}
		byAccount[internalAccID] = append(byAccount[internalAccID], action)
	}

	for _, plaidTxnID := range removed {
		action, accID, err := p.buildRemoveAction(ctx, reader, plaidTxnID, plaidToInternal)
		if err != nil {
			return err
		}
		if action != nil {
			byAccount[accID] = append(byAccount[accID], action)
		}
	}

	balances, err := p.client.GetAccountBalances(ctx, item.AccessToken)
	if err != nil {
		return fmt.Errorf("fetching account balances for item %s: %w", item.ID, err)
	}
	for _, b := range balances {
		internalAccID, ok := plaidToInternal[b.PlaidAccountID]
		if !ok || !b.HasCurrentBalance {
			continue
		}
		bal, err := amountFromPlaidFloat(b.CurrentBalance)
		if err != nil {
			return fmt.Errorf("plaid balance for account %s: %w", b.PlaidAccountID, err)
		}
		byAccount[internalAccID] = append(byAccount[internalAccID], &actions.PlaidSetAccountBalance{
			AccountID: internalAccID,
			Balance:   bal,
		})
	}

	*onSuccess = append(*onSuccess, &actions.PlaidUpdateCursor{
		ItemID:     item.ID,
		NextCursor: nextCursor,
	})

	return nil
}

const plaidDateLayout = "2006-01-02"

func parsePlaidDate(dateStr string) (time.Time, error) {
	t, err := time.Parse(plaidDateLayout, dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse plaid date %q: %w", dateStr, err)
	}
	return t, nil
}

// amountFromPlaidFloat turns Plaid's JSON float amount into a decimal without an extra float64 round-trip.
func amountFromPlaidFloat(f float64) (decimal.Decimal, error) {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("decimal from plaid amount %g: %w", f, err)
	}
	return d, nil
}

func (p *PlaidProvider) buildAddAction(t plaidclient.SyncTransaction, accountID uuid.UUID) (actions.IAction, error) {
	txnID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("generating UUID: %w", err)
	}
	date, err := parsePlaidDate(t.Date)
	if err != nil {
		return nil, err
	}
	amount, err := amountFromPlaidFloat(t.Amount)
	if err != nil {
		return nil, err
	}
	amount = amount.Neg()

	return &actions.PlaidAddTransaction{
		TransactionID:      txnID,
		AccountID:          accountID,
		PlaidTransactionID: t.PlaidTransactionID,
		PlaidAccountID:     t.PlaidAccountID,
		Amount:             amount,
		Name:               t.Name,
		Date:               date,
	}, nil
}

func (p *PlaidProvider) buildModifyAction(ctx context.Context, reader *storage.Reader, t plaidclient.SyncTransaction, accountID uuid.UUID) (actions.IAction, error) {
	link, err := reader.Plaid.FindTransactionLink(ctx, t.PlaidTransactionID)
	if err != nil {
		return nil, fmt.Errorf("finding transaction link for %s: %w", t.PlaidTransactionID, err)
	}
	if link == nil {
		// Modified transaction we never saw — treat as an add
		return p.buildAddAction(t, accountID)
	}

	date, err := parsePlaidDate(t.Date)
	if err != nil {
		return nil, err
	}
	amount, err := amountFromPlaidFloat(t.Amount)
	if err != nil {
		return nil, err
	}
	amount = amount.Neg()
	return &actions.PlaidModifyTransaction{
		TransactionID: link.TransactionID,
		AccountID:     accountID,
		Amount:        amount,
		Name:          t.Name,
		Date:          date,
	}, nil
}

func (p *PlaidProvider) buildRemoveAction(ctx context.Context, reader *storage.Reader, plaidTxnID string, plaidToInternal map[string]uuid.UUID) (actions.IAction, uuid.UUID, error) {
	link, err := reader.Plaid.FindTransactionLink(ctx, plaidTxnID)
	if err != nil {
		return nil, uuid.UUID{}, fmt.Errorf("finding transaction link for removal %s: %w", plaidTxnID, err)
	}
	if link == nil {
		return nil, uuid.UUID{}, nil // already gone or never synced
	}
	internalAccID, ok := plaidToInternal[link.PlaidAccountID]
	if !ok {
		return nil, uuid.UUID{}, nil
	}
	return &actions.PlaidRemoveTransaction{
		TransactionID:      link.TransactionID,
		AccountID:          internalAccID,
		PlaidTransactionID: plaidTxnID,
	}, internalAccID, nil
}

func (p *PlaidProvider) fetchAllPages(ctx context.Context, item *plaidstore.PlaidItem) (
	added []plaidclient.SyncTransaction,
	modified []plaidclient.SyncTransaction,
	removed []string,
	nextCursor string,
	err error,
) {
	syncCtx := ctx
	if item.Cursor == "" {
		var cancel context.CancelFunc
		syncCtx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	const maxPages = 1000
	cursor := item.Cursor
	for page := range maxPages {
		result, syncErr := p.client.SyncTransactions(syncCtx, item.AccessToken, cursor)
		if syncErr != nil {
			return nil, nil, nil, "", syncErr
		}

		for _, t := range result.Added {
			if !t.Pending {
				added = append(added, t)
			}
		}
		for _, t := range result.Modified {
			if !t.Pending {
				modified = append(modified, t)
			}
		}
		removed = append(removed, result.Removed...)

		cursor = result.NextCursor
		if !result.HasMore {
			break
		}
		if page == maxPages-1 {
			return nil, nil, nil, "", fmt.Errorf("plaid sync exceeded %d pages for item %s", maxPages, item.ID)
		}
	}

	return added, modified, removed, cursor, nil
}
