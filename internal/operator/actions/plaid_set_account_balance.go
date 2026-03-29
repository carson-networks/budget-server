package actions

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/storage"
)

type PlaidSetAccountBalance struct {
	AccountID uuid.UUID
	Balance   decimal.Decimal
}

func (a *PlaidSetAccountBalance) Perform(ctx context.Context, writer *storage.Writer) error {
	if err := writer.Account.UpdateBalance(ctx, a.AccountID, a.Balance); err != nil {
		return fmt.Errorf("setting account balance from plaid: %w", err)
	}
	return nil
}
