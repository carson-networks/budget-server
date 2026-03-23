package account

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
)

// SyncAccountsBody is the request body for POST /v1/accounts/sync.
type SyncAccountsBody struct {
	AccountIDs []string `json:"accountIDs,omitempty" doc:"Optional list of internal account UUIDs to sync. Empty syncs all Plaid-linked accounts."`
}

// SyncAccountsInput is the Huma input for the sync endpoint.
type SyncAccountsInput struct {
	Body SyncAccountsBody
}

// SyncAccountsOutput is the Huma output for the sync endpoint.
type SyncAccountsOutput struct {
	Body struct {
		SyncedAccounts      int `json:"syncedAccounts" doc:"Number of Plaid Items (institution connections) synced"`
		AddedTransactions   int `json:"addedTransactions" doc:"Number of transactions added"`
		RemovedTransactions int `json:"removedTransactions" doc:"Number of transactions removed"`
	}
}

// syncPlaidReader is the subset of plaid.Reader the sync handler needs.
type syncPlaidReader interface {
	ListItemsForSync(ctx context.Context, accountIDs []uuid.UUID) ([]*plaidstore.PlaidItem, error)
}

// plaidSyncer is the subset of plaid.Client the sync handler needs.
type plaidSyncer interface {
	SyncTransactions(ctx context.Context, accessToken, cursor string) (*plaidclient.SyncResult, error)
}

// SyncAccountsHandler handles POST /v1/accounts/sync.
type SyncAccountsHandler struct {
	Operator    operator.IProcessor
	PlaidClient plaidSyncer
	PlaidReader syncPlaidReader
}

func NewSyncAccountsHandler(op operator.IProcessor, client *plaidclient.Client, store *storage.Storage) *SyncAccountsHandler {
	return &SyncAccountsHandler{Operator: op, PlaidClient: client, PlaidReader: store.Read().Plaid}
}

func (h *SyncAccountsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "sync-plaid-accounts",
		Method:      http.MethodPost,
		Path:        "/v1/accounts/sync",
		Summary:     "Sync Plaid accounts",
		Description: "Pulls the latest transactions from Plaid for all linked accounts, or for the specified accounts.",
		Tags:        []string{"Accounts", "Plaid"},
	}, h.handle)
}

func (h *SyncAccountsHandler) handle(ctx context.Context, input *SyncAccountsInput) (*SyncAccountsOutput, error) {
	accountUUIDs := make([]uuid.UUID, 0, len(input.Body.AccountIDs))
	for _, idStr := range input.Body.AccountIDs {
		id, err := uuid.FromString(idStr)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, fmt.Sprintf("invalid account ID: %s", idStr))
		}
		accountUUIDs = append(accountUUIDs, id)
	}

	items, err := h.PlaidReader.ListItemsForSync(ctx, accountUUIDs)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list Plaid items", err)
	}
	if len(items) == 0 {
		return &SyncAccountsOutput{}, nil
	}

	out := &SyncAccountsOutput{}
	syncItems := make([]actions.SyncedItem, 0, len(items))
	for _, item := range items {
		added, removed, nextCursor, err := h.fetchAllSyncPages(ctx, item)
		if err != nil {
			return nil, huma.NewError(http.StatusInternalServerError, "Plaid sync failed", err)
		}
		syncItems = append(syncItems, actions.SyncedItem{
			ItemID:       item.ID,
			Transactions: append(added, removed...),
			NextCursor:   nextCursor,
		})
		out.Body.SyncedAccounts++
		out.Body.AddedTransactions += len(added)
		out.Body.RemovedTransactions += len(removed)
	}

	if err := h.Operator.Process(ctx, &actions.SyncPlaidAccounts{Items: syncItems}); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to apply Plaid sync", err)
	}

	return out, nil
}

// fetchAllSyncPages paginates through Plaid's /transactions/sync until HasMore=false.
func (h *SyncAccountsHandler) fetchAllSyncPages(ctx context.Context, item *plaidstore.PlaidItem) (added []actions.SyncedTransaction, removed []actions.SyncedTransaction, nextCursor string, err error) {
	syncCtx := ctx
	if item.Cursor == "" {
		var cancel context.CancelFunc
		syncCtx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()
	}

	const maxPages = 1000
	cursor := item.Cursor
	for page := range maxPages {
		result, syncErr := h.PlaidClient.SyncTransactions(syncCtx, item.AccessToken, cursor)
		if syncErr != nil {
			return nil, nil, "", syncErr
		}

		for _, t := range result.Added {
			if t.Pending {
				continue
			}
			date, _ := time.Parse("2006-01-02", t.Date)
			amount := decimal.NewFromFloat(t.Amount).Neg()
			added = append(added, actions.SyncedTransaction{
				PlaidTransactionID: t.PlaidTransactionID,
				PlaidAccountID:     t.PlaidAccountID,
				Amount:             amount,
				Name:               t.Name,
				Date:               date,
			})
		}

		for _, t := range result.Modified {
			if t.Pending {
				continue
			}
			date, _ := time.Parse("2006-01-02", t.Date)
			amount := decimal.NewFromFloat(t.Amount).Neg()
			added = append(added, actions.SyncedTransaction{
				PlaidTransactionID: t.PlaidTransactionID,
				PlaidAccountID:     t.PlaidAccountID,
				Amount:             amount,
				Name:               t.Name,
				Date:               date,
			})
		}

		for _, plaidTxnID := range result.Removed {
			removed = append(removed, actions.SyncedTransaction{
				PlaidTransactionID: plaidTxnID,
				IsRemoved:          true,
			})
		}

		cursor = result.NextCursor
		if !result.HasMore {
			break
		}
		if page == maxPages-1 {
			return nil, nil, "", fmt.Errorf("plaid sync exceeded %d pages for item %s", maxPages, item.ID)
		}
	}

	return added, removed, cursor, nil
}
