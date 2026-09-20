package actions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

type UpdateAccount struct {
	ID              uuid.UUID
	Name            *string
	SubType         *string
	StartingBalance *decimal.Decimal

	IAction
}

func (u *UpdateAccount) Perform(ctx context.Context, writer *storage.Writer) error {
	existing, err := writer.Account.FindByIDForUpdate(ctx, u.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}
		return err
	}
	if existing == nil {
		return ErrAccountNotFound
	}

	update := &account.AccountUpdate{
		Name:            u.Name,
		SubType:         u.SubType,
		StartingBalance: u.StartingBalance,
	}
	return writer.Account.Update(ctx, u.ID, update)
}
