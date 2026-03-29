package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/storage"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

type PlaidAddTransaction struct {
	TransactionID      uuid.UUID
	AccountID          uuid.UUID
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             decimal.Decimal
	Name               string
	Date               time.Time
}

func (a *PlaidAddTransaction) Perform(ctx context.Context, writer *storage.Writer) error {
	_, err := writer.Transaction.Insert(ctx, &transaction.TransactionCreate{
		ID:              &a.TransactionID,
		AccountID:       a.AccountID,
		CategoryID:      nil,
		Amount:          a.Amount,
		TransactionName: a.Name,
		TransactionDate: a.Date,
	})
	if err != nil {
		return fmt.Errorf("inserting transaction: %w", err)
	}

	if err := writer.Plaid.CreateTransactionLink(ctx, &plaidstore.TransactionLink{
		PlaidTransactionID: a.PlaidTransactionID,
		TransactionID:      a.TransactionID,
		PlaidAccountID:     a.PlaidAccountID,
	}); err != nil {
		return fmt.Errorf("creating transaction link: %w", err)
	}

	return nil
}
