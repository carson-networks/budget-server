package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage/account"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
)

func validLinkAction() *LinkPlaidItem {
	return &LinkPlaidItem{
		AccessToken:     "access-sandbox-abc",
		PlaidItemID:     "plaid-item-123",
		InstitutionID:   "ins_1",
		InstitutionName: "Chase",
		Accounts: []PlaidAccountToLink{
			{
				PlaidAccountID: "plaid-acc-1",
				Name:           "Checking",
				Type:           account.AccountType(0),
				SubType:        "checking",
				Balance:        decimal.NewFromFloat(500.00),
			},
		},
	}
}

func TestLinkPlaidItem_Perform_Success_SingleAccount(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	a := validLinkAction()

	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, &plaidstore.PlaidItemCreate{
		AccessToken:     a.AccessToken,
		PlaidItemID:     a.PlaidItemID,
		InstitutionID:   a.InstitutionID,
		InstitutionName: a.InstitutionName,
	}).Return(itemID, nil)
	mockAccount.On("Create", mock.Anything, "Checking", account.AccountType(0), "checking", decimal.NewFromFloat(500.00)).
		Return(accID, nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, &plaidstore.AccountLink{
		PlaidAccountID: "plaid-acc-1",
		AccountID:      accID,
		PlaidItemID:    itemID,
	}).Return(nil)

	err := a.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestLinkPlaidItem_Perform_Success_MultipleAccounts(t *testing.T) {
	itemID := mustID(t)
	acc1ID := mustID(t)
	acc2ID := mustID(t)

	a := &LinkPlaidItem{
		AccessToken:     "access-sandbox-abc",
		PlaidItemID:     "plaid-item-456",
		InstitutionID:   "ins_2",
		InstitutionName: "Bank of America",
		Accounts: []PlaidAccountToLink{
			{PlaidAccountID: "p-acc-1", Name: "Checking", Type: account.AccountType(0), SubType: "checking", Balance: decimal.NewFromFloat(1000.00)},
			{PlaidAccountID: "p-acc-2", Name: "Savings", Type: account.AccountType(0), SubType: "savings", Balance: decimal.NewFromFloat(5000.00)},
		},
	}

	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, &plaidstore.PlaidItemCreate{
		AccessToken: a.AccessToken, PlaidItemID: a.PlaidItemID,
		InstitutionID: a.InstitutionID, InstitutionName: a.InstitutionName,
	}).Return(itemID, nil)

	mockAccount.On("Create", mock.Anything, "Checking", account.AccountType(0), "checking", decimal.NewFromFloat(1000.00)).Return(acc1ID, nil)
	mockAccount.On("Create", mock.Anything, "Savings", account.AccountType(0), "savings", decimal.NewFromFloat(5000.00)).Return(acc2ID, nil)

	mockPlaid.On("CreateAccountLink", mock.Anything, &plaidstore.AccountLink{PlaidAccountID: "p-acc-1", AccountID: acc1ID, PlaidItemID: itemID}).Return(nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, &plaidstore.AccountLink{PlaidAccountID: "p-acc-2", AccountID: acc2ID, PlaidItemID: itemID}).Return(nil)

	err := a.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestLinkPlaidItem_Perform_CreateItemError(t *testing.T) {
	a := validLinkAction()
	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(mustID(t), errors.New("db error"))

	err := a.Perform(context.Background(), w)
	assert.Error(t, err)
	mockAccount.AssertNotCalled(t, "Create")
	mockPlaid.AssertNotCalled(t, "CreateAccountLink")
}

func TestLinkPlaidItem_Perform_CreateAccountError(t *testing.T) {
	itemID := mustID(t)
	a := validLinkAction()
	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockAccount.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(mustID(t), errors.New("insert failed"))

	err := a.Perform(context.Background(), w)
	assert.Error(t, err)
	mockPlaid.AssertNotCalled(t, "CreateAccountLink")
}

func TestLinkPlaidItem_Perform_CreateAccountLinkError(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	a := validLinkAction()
	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockAccount.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(accID, nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, mock.Anything).Return(errors.New("link failed"))

	err := a.Perform(context.Background(), w)
	assert.Error(t, err)
}

func TestLinkPlaidItem_Perform_MultipleAccounts_StopsOnFirstAccountError(t *testing.T) {
	itemID := mustID(t)
	acc1ID := mustID(t)

	a := &LinkPlaidItem{
		AccessToken: "at", PlaidItemID: "pid", InstitutionID: "ins", InstitutionName: "Bank",
		Accounts: []PlaidAccountToLink{
			{PlaidAccountID: "p-1", Name: "Checking", Balance: decimal.NewFromFloat(100)},
			{PlaidAccountID: "p-2", Name: "Savings", Balance: decimal.NewFromFloat(200)},
		},
	}

	w, mockAccount, _, mockPlaid := newSyncWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, mock.Anything).Return(nil)
	// First account succeeds, second fails.
	mockAccount.On("Create", mock.Anything, "Checking", mock.Anything, mock.Anything, mock.Anything).Return(acc1ID, nil)
	mockAccount.On("Create", mock.Anything, "Savings", mock.Anything, mock.Anything, mock.Anything).Return(mustID(t), errors.New("create failed"))

	err := a.Perform(context.Background(), w)
	assert.Error(t, err)
	// Only one account link should have been created (for the first account).
	mockPlaid.AssertNumberOfCalls(t, "CreateAccountLink", 1)
}
