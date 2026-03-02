package actions

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/shopspring/decimal"
)

type CreateAccount struct {
	Name            string
	Type            account.AccountType
	SubType         string
	StartingBalance decimal.Decimal

	IAction
}

func (c *CreateAccount) Perform(ctx context.Context, writer *storage.Writer) error {
	err := writer.Account.Create(ctx, c.Name, c.Type, c.SubType, c.StartingBalance)
	if err != nil {
		return err
	}

	return nil
}
