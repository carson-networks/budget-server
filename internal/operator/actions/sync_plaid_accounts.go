package actions

import (
	"context"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// SyncedTransaction is one transaction from Plaid's /transactions/sync, already
// converted to server conventions (pending filtered out, amount negated).
type SyncedTransaction struct {
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             decimal.Decimal // server convention: negative = debit, positive = credit
	Name               string
	Date               time.Time
	IsRemoved          bool
}

// SyncedItem holds all pre-fetched Plaid data for one institution connection (Item).
type SyncedItem struct {
	ItemID       uuid.UUID
	Transactions []SyncedTransaction // pending already filtered out by handler
	NextCursor   string
}

// SyncPlaidAccounts applies Plaid transaction updates for all Items in a single DB transaction.
// All Plaid API calls must be completed by the handler before enqueueing this action.
type SyncPlaidAccounts struct {
	Items []SyncedItem
	IAction
}

func (a *SyncPlaidAccounts) Perform(ctx context.Context, writer *storage.Writer) error {
	for _, item := range a.Items {
		if err := syncItem(ctx, writer, item); err != nil {
			return err
		}
	}
	return nil
}

func syncItem(ctx context.Context, writer *storage.Writer, item SyncedItem) error {
	links, err := writer.Plaid.ListAccountLinksByItemID(ctx, item.ItemID)
	if err != nil {
		return err
	}
	plaidToInternal := make(map[string]uuid.UUID, len(links))
	for _, link := range links {
		plaidToInternal[link.PlaidAccountID] = link.AccountID
	}

	for _, t := range item.Transactions {
		internalAccID, ok := plaidToInternal[t.PlaidAccountID]
		if !ok {
			// This Plaid account is not tracked in this server; skip it.
			continue
		}

		if t.IsRemoved {
			if err := removeTransaction(ctx, writer, t.PlaidTransactionID); err != nil {
				return err
			}
			continue
		}

		if err := upsertTransaction(ctx, writer, t, internalAccID); err != nil {
			return err
		}
	}

	return writer.Plaid.UpdateCursor(ctx, item.ItemID, item.NextCursor)
}

func removeTransaction(ctx context.Context, writer *storage.Writer, plaidTxnID string) error {
	link, err := writer.Plaid.FindTransactionLink(ctx, plaidTxnID)
	if err != nil {
		return err
	}
	if link == nil {
		return nil // already gone or never synced
	}

	deleted, err := writer.Transaction.Delete(ctx, link.TransactionID)
	if err != nil {
		return err
	}
	if err := writer.Plaid.DeleteTransactionLink(ctx, plaidTxnID); err != nil {
		return err
	}
	if deleted != nil {
		// Reverse the balance effect of the deleted transaction
		acc, err := writer.Account.FindByIDForUpdate(ctx, deleted.AccountID)
		if err != nil {
			return err
		}
		return writer.Account.UpdateBalance(ctx, deleted.AccountID, acc.Balance.Sub(deleted.Amount))
	}
	return nil
}

func upsertTransaction(ctx context.Context, writer *storage.Writer, t SyncedTransaction, internalAccID uuid.UUID) error {
	existing, err := writer.Plaid.FindTransactionLink(ctx, t.PlaidTransactionID)
	if err != nil {
		return err
	}

	if existing != nil {
		// Modified transaction
		old, err := writer.Transaction.FindByID(ctx, existing.TransactionID)
		if err != nil {
			return err
		}
		if old == nil {
			// Transaction was deleted externally; remove the stale link and fall through to insert.
			_ = writer.Plaid.DeleteTransactionLink(ctx, t.PlaidTransactionID)
		} else {
			if err := writer.Transaction.Update(ctx, existing.TransactionID, &transaction.TransactionUpdate{
				Amount:          t.Amount,
				TransactionName: t.Name,
				TransactionDate: t.Date,
			}); err != nil {
				return err
			}
			acc, err := writer.Account.FindByIDForUpdate(ctx, internalAccID)
			if err != nil {
				return err
			}
			delta := t.Amount.Sub(old.Amount)
			return writer.Account.UpdateBalance(ctx, internalAccID, acc.Balance.Add(delta))
		}
	}

	// New transaction — no category; user can assign one in the UI
	txnID, err := writer.Transaction.Insert(ctx, &transaction.TransactionCreate{
		AccountID:       internalAccID,
		CategoryID:      nil,
		Amount:          t.Amount,
		TransactionName: t.Name,
		TransactionDate: t.Date,
	})
	if err != nil {
		return err
	}
	if err := writer.Plaid.CreateTransactionLink(ctx, &plaidstore.TransactionLink{
		PlaidTransactionID: t.PlaidTransactionID,
		TransactionID:      txnID,
		PlaidAccountID:     t.PlaidAccountID,
	}); err != nil {
		return err
	}
	acc, err := writer.Account.FindByIDForUpdate(ctx, internalAccID)
	if err != nil {
		return err
	}
	return writer.Account.UpdateBalance(ctx, internalAccID, acc.Balance.Add(t.Amount))
}
