package actions

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

var ErrTransactionNotFound = errors.New("transaction not found")

type UpdateTransaction struct {
	ID              uuid.UUID
	CategoryID      *uuid.UUID
	Amount          *decimal.Decimal
	TransactionName *string
	MerchantName    *string
	TransactionDate *time.Time
}

func (u *UpdateTransaction) Perform(ctx context.Context, writer *storage.Writer) error {
	existing, err := writer.Transaction.FindByIDForUpdate(ctx, u.ID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && existing == nil) {
		return ErrTransactionNotFound
	}
	if err != nil {
		return err
	}
	if u.CategoryID != nil {
		category, err := writer.Category.GetByID(ctx, *u.CategoryID)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && category == nil) {
			return ErrCategoryNotFoundForTransaction
		}
		if err != nil {
			return err
		}
		if category.IsDisabled {
			return ErrCategoryDisabled
		}
		if category.IsParent {
			return ErrCategoryIsParent
		}
	}
	if err := writer.Transaction.Patch(ctx, u.ID, &transaction.TransactionPatch{
		CategoryID: u.CategoryID, Amount: u.Amount, TransactionName: u.TransactionName,
		MerchantName: u.MerchantName, TransactionDate: u.TransactionDate,
	}); err != nil {
		return err
	}
	if u.Amount != nil {
		delta := u.Amount.Sub(existing.Amount)
		if !delta.IsZero() {
			if err := writer.Account.AdjustBalance(ctx, existing.AccountID, delta); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return ErrAccountNotFound
				}
				return err
			}
		}
	}
	return nil
}
