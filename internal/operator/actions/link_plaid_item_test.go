package actions

import (
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
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

func newLinkTestWriter(t *testing.T) (*storage.Writer, *storage.MockIAccountWriter, *storage.MockIPlaidWriter, *storage.MockISyncWriter) {
	t.Helper()
	w := storage.NewWriterForTest()
	return w, w.Account.(*storage.MockIAccountWriter), w.Plaid.(*storage.MockIPlaidWriter), w.Sync.(*storage.MockISyncWriter)
}

func TestLinkPlaidItem_Perform_Success_SingleAccount(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	accID := uuid.Must(uuid.NewV4())
	a := validLinkAction()

	w, mockAccount, mockPlaid, mockSync := newLinkTestWriter(t)

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
	mockSync.On("Create", mock.Anything, accID, syncstore.SyncType_Plaid).Return(nil)

	require.NoError(t, a.Perform(context.Background(), w))
	assert.Equal(t, []uuid.UUID{accID}, a.CreatedAccountIDs)
	mockPlaid.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
	mockSync.AssertExpectations(t)
}

func TestLinkPlaidItem_Perform_Success_MultipleAccounts(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	acc1ID := uuid.Must(uuid.NewV4())
	acc2ID := uuid.Must(uuid.NewV4())

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

	w, mockAccount, mockPlaid, mockSync := newLinkTestWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, &plaidstore.PlaidItemCreate{
		AccessToken: a.AccessToken, PlaidItemID: a.PlaidItemID,
		InstitutionID: a.InstitutionID, InstitutionName: a.InstitutionName,
	}).Return(itemID, nil)

	mockAccount.On("Create", mock.Anything, "Checking", account.AccountType(0), "checking", decimal.NewFromFloat(1000.00)).Return(acc1ID, nil)
	mockAccount.On("Create", mock.Anything, "Savings", account.AccountType(0), "savings", decimal.NewFromFloat(5000.00)).Return(acc2ID, nil)

	mockPlaid.On("CreateAccountLink", mock.Anything, &plaidstore.AccountLink{PlaidAccountID: "p-acc-1", AccountID: acc1ID, PlaidItemID: itemID}).Return(nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, &plaidstore.AccountLink{PlaidAccountID: "p-acc-2", AccountID: acc2ID, PlaidItemID: itemID}).Return(nil)

	mockSync.On("Create", mock.Anything, mock.Anything, syncstore.SyncType_Plaid).Return(nil)

	require.NoError(t, a.Perform(context.Background(), w))
	assert.Equal(t, []uuid.UUID{acc1ID, acc2ID}, a.CreatedAccountIDs)
	mockPlaid.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
	mockSync.AssertNumberOfCalls(t, "Create", 2)
}

func TestLinkPlaidItem_Perform_CreateItemError(t *testing.T) {
	a := validLinkAction()
	w, mockAccount, mockPlaid, _ := newLinkTestWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(uuid.Must(uuid.NewV4()), errors.New("db error"))

	assert.Error(t, a.Perform(context.Background(), w))
	mockAccount.AssertNotCalled(t, "Create")
	mockPlaid.AssertNotCalled(t, "CreateAccountLink")
}

func TestLinkPlaidItem_Perform_CreateAccountError(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	a := validLinkAction()
	w, mockAccount, mockPlaid, _ := newLinkTestWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockAccount.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(uuid.Must(uuid.NewV4()), errors.New("insert failed"))

	assert.Error(t, a.Perform(context.Background(), w))
	mockPlaid.AssertNotCalled(t, "CreateAccountLink")
}

func TestLinkPlaidItem_Perform_CreateAccountLinkError(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	accID := uuid.Must(uuid.NewV4())
	a := validLinkAction()
	w, mockAccount, mockPlaid, _ := newLinkTestWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockAccount.On("Create", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(accID, nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, mock.Anything).Return(errors.New("link failed"))

	assert.Error(t, a.Perform(context.Background(), w))
}

func TestLinkPlaidItem_Perform_MultipleAccounts_StopsOnFirstAccountError(t *testing.T) {
	itemID := uuid.Must(uuid.NewV4())
	acc1ID := uuid.Must(uuid.NewV4())

	a := &LinkPlaidItem{
		AccessToken: "at", PlaidItemID: "pid", InstitutionID: "ins", InstitutionName: "Bank",
		Accounts: []PlaidAccountToLink{
			{PlaidAccountID: "p-1", Name: "Checking", Balance: decimal.NewFromFloat(100)},
			{PlaidAccountID: "p-2", Name: "Savings", Balance: decimal.NewFromFloat(200)},
		},
	}

	w, mockAccount, mockPlaid, mockSync := newLinkTestWriter(t)

	mockPlaid.On("CreateItem", mock.Anything, mock.Anything).Return(itemID, nil)
	mockPlaid.On("CreateAccountLink", mock.Anything, mock.Anything).Return(nil)
	mockSync.On("Create", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockAccount.On("Create", mock.Anything, "Checking", mock.Anything, mock.Anything, mock.Anything).Return(acc1ID, nil)
	mockAccount.On("Create", mock.Anything, "Savings", mock.Anything, mock.Anything, mock.Anything).Return(uuid.Must(uuid.NewV4()), errors.New("create failed"))

	assert.Error(t, a.Perform(context.Background(), w))
	mockPlaid.AssertNumberOfCalls(t, "CreateAccountLink", 1)
	mockSync.AssertNumberOfCalls(t, "Create", 1)
}
