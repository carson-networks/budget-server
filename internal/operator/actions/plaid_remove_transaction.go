package actions

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/storage"
)

type PlaidRemoveTransaction struct {
	TransactionID      uuid.UUID
	AccountID          uuid.UUID
	PlaidTransactionID string
}

func (a *PlaidRemoveTransaction) Perform(ctx context.Context, writer *storage.Writer) error {
	if _, err := writer.Transaction.Delete(ctx, a.TransactionID); err != nil {
		return fmt.Errorf("deleting transaction: %w", err)
	}

	if err := writer.Plaid.DeleteTransactionLink(ctx, a.PlaidTransactionID); err != nil {
		return fmt.Errorf("deleting transaction link: %w", err)
	}

	return nil
}
