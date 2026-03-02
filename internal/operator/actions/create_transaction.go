package actions

import (
	"context"
	"errors"
	"time"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

type CreateTransaction struct {
	AccountID       uuid.UUID
	CategoryID      uuid.UUID
	Amount          decimal.Decimal
	TransactionName string
	TransactionDate time.Time
	IAction
}

func (t *CreateTransaction) Perform(ctx context.Context, writer *storage.Writer) error {
	account, err := writer.Account.FindByIDForUpdate(ctx, t.AccountID)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.New("account not found")
	}

	storageCreate := &transaction.TransactionCreate{
		AccountID:       t.AccountID,
		CategoryID:      t.CategoryID,
		Amount:          t.Amount,
		TransactionName: t.TransactionName,
		TransactionDate: t.TransactionDate,
	}
	_, err = writer.Transaction.Insert(ctx, storageCreate)
	if err != nil {
		return err
	}

	newBalance := account.Balance.Add(t.Amount)
	err = writer.Account.UpdateBalance(ctx, t.AccountID, newBalance)
	if err != nil {
		return err
	}

	return nil
}
