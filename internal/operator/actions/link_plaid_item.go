package actions

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/shopspring/decimal"
)

// PlaidAccountToLink describes a Plaid account the user has selected to import.
type PlaidAccountToLink struct {
	PlaidAccountID string
	Name           string
	Type           account.AccountType
	SubType        string
	Balance        decimal.Decimal // used as starting balance
}

// LinkPlaidItem stores the Plaid Item in the DB and creates a budget account for each
// selected Plaid account. This action runs inside the operator transaction; all Plaid
// API calls must be made by the handler before enqueueing.
type LinkPlaidItem struct {
	AccessToken     string
	PlaidItemID     string
	InstitutionID   string
	InstitutionName string
	Accounts        []PlaidAccountToLink

	IAction
}

func (a *LinkPlaidItem) Perform(ctx context.Context, writer *storage.Writer) error {
	// 1. Create the Plaid item record
	itemID, err := writer.Plaid.CreateItem(ctx, &plaidstore.PlaidItemCreate{
		AccessToken:     a.AccessToken,
		PlaidItemID:     a.PlaidItemID,
		InstitutionID:   a.InstitutionID,
		InstitutionName: a.InstitutionName,
	})
	if err != nil {
		return err
	}

	// 2. Create a budget account and account link for each selected Plaid account
	for _, acc := range a.Accounts {
		accountID, err := writer.Account.Create(ctx, acc.Name, acc.Type, acc.SubType, acc.Balance)
		if err != nil {
			return err
		}
		if err := writer.Plaid.CreateAccountLink(ctx, &plaidstore.AccountLink{
			PlaidAccountID: acc.PlaidAccountID,
			AccountID:      accountID,
			PlaidItemID:    itemID,
		}); err != nil {
			return err
		}
	}

	return nil
}
