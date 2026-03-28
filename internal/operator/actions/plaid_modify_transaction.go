package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

type PlaidModifyTransaction struct {
	TransactionID uuid.UUID
	AccountID     uuid.UUID
	Amount        decimal.Decimal
	Name          string
	Date          time.Time
}

func (a *PlaidModifyTransaction) Perform(ctx context.Context, writer *storage.Writer) error {
	old, err := writer.Transaction.FindByID(ctx, a.TransactionID)
	if err != nil {
		return fmt.Errorf("finding transaction for update: %w", err)
	}
	if old == nil {
		return fmt.Errorf("transaction %s not found", a.TransactionID)
	}

	if err := writer.Transaction.Update(ctx, a.TransactionID, &transaction.TransactionUpdate{
		Amount:          a.Amount,
		TransactionName: a.Name,
		TransactionDate: a.Date,
	}); err != nil {
		return fmt.Errorf("updating transaction: %w", err)
	}

	return nil
}
