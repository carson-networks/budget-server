package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
)

// plaidClient is the subset of plaid.Client the provider needs.
type plaidClient interface {
	SyncTransactions(ctx context.Context, accessToken, cursor string) (*plaidclient.SyncResult, error)
	GetAccountBalances(ctx context.Context, accessToken string) ([]plaidclient.AccountBalance, error)
}

// PlaidProvider implements sync.Provider for Plaid-linked accounts.
type PlaidProvider struct {
	client plaidClient
}

func NewPlaidProvider(client plaidClient) *PlaidProvider {
	return &PlaidProvider{client: client}
}

func (p *PlaidProvider) Type() syncstore.SyncType {
	return syncstore.SyncType_Plaid
}

func (p *PlaidProvider) Sync(ctx context.Context, reader *storage.Reader, accountIDs []uuid.UUID) (map[uuid.UUID][]actions.IAction, error) {
	items, err := reader.Plaid.ListItemsForSync(ctx, accountIDs)
	if err != nil {
		return nil, fmt.Errorf("listing plaid items: %w", err)
	}
	if len(items) == 0 {
		return nil, nil
	}

	result := make(map[uuid.UUID][]actions.IAction)

	for _, item := range items {
		if err := p.syncItem(ctx, reader, item, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (p *PlaidProvider) syncItem(ctx context.Context, reader *storage.Reader, item *plaidstore.PlaidItem, result map[uuid.UUID][]actions.IAction) error {
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

	// Build add actions
	for _, t := range added {
		internalAccID, ok := plaidToInternal[t.PlaidAccountID]
		if !ok {
			continue
		}
		action, err := p.buildAddAction(t, internalAccID)
		if err != nil {
			return err
		}
		result[internalAccID] = append(result[internalAccID], action)
	}

	// Build modify actions
	for _, t := range modified {
		internalAccID, ok := plaidToInternal[t.PlaidAccountID]
		if !ok {
			continue
		}
		action, err := p.buildModifyAction(ctx, reader, t, internalAccID)
		if err != nil {
			return err
		}
		result[internalAccID] = append(result[internalAccID], action)
	}

	// Build remove actions
	for _, plaidTxnID := range removed {
		action, accID, err := p.buildRemoveAction(ctx, reader, plaidTxnID, plaidToInternal)
		if err != nil {
			return err
		}
		if action != nil {
			result[accID] = append(result[accID], action)
		}
	}

	// Fetch current balances from Plaid and emit a balance-set action per linked account.
	balances, err := p.client.GetAccountBalances(ctx, item.AccessToken)
	if err != nil {
		return fmt.Errorf("fetching account balances for item %s: %w", item.ID, err)
	}
	for _, b := range balances {
		internalAccID, ok := plaidToInternal[b.PlaidAccountID]
		if !ok {
			continue
		}
		result[internalAccID] = append(result[internalAccID], &actions.PlaidSetAccountBalance{
			AccountID: internalAccID,
			Balance:   decimal.NewFromFloat(b.CurrentBalance),
		})
	}

	// Append cursor update for each account that has actions for this item.
	for _, link := range links {
		if _, hasActions := result[link.AccountID]; hasActions {
			result[link.AccountID] = append(result[link.AccountID], &actions.PlaidUpdateCursor{
				ItemID:     item.ID,
				NextCursor: nextCursor,
			})
		}
	}

	return nil
}

func (p *PlaidProvider) buildAddAction(t plaidclient.SyncTransaction, accountID uuid.UUID) (actions.IAction, error) {
	txnID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("generating UUID: %w", err)
	}
	date, _ := time.Parse("2006-01-02", t.Date)
	amount := decimal.NewFromFloat(t.Amount).Neg()

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

	date, _ := time.Parse("2006-01-02", t.Date)
	amount := decimal.NewFromFloat(t.Amount).Neg()
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

// fetchAllPages paginates through Plaid's /transactions/sync until HasMore=false.
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
