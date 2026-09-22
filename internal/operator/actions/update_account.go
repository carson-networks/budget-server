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
	// Create seeds balance from starting_balance; transactions only move balance.
	// When starting_balance changes, shift current balance by the same delta so
	// prior activity is preserved relative to the corrected opening balance.
	if u.StartingBalance != nil {
		delta := u.StartingBalance.Sub(existing.StartingBalance)
		newBalance := existing.Balance.Add(delta)
		update.Balance = &newBalance
	}
	return writer.Account.Update(ctx, u.ID, update)
}
