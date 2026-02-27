package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig"
)

func newAccountTestService(t *testing.T) (*AccountService, *sqlconfig.MockIAccountTable) {
	t.Helper()
	mockTable := sqlconfig.NewMockIAccountTable(t)
	store := &storage.Storage{Accounts: mockTable}
	svc := NewAccountService(store)
	return svc, mockTable
}

func makeStorageAccounts(n int, createdAt time.Time) []*sqlconfig.Account {
	rows := make([]*sqlconfig.Account, n)
	for i := range rows {
		rows[i] = &sqlconfig.Account{
			ID:              uuid.Must(uuid.NewV4()),
			Name:            "Checking",
			Type:            sqlconfig.AccountTypeCash,
			SubType:         "Primary",
			Balance:         decimal.RequireFromString("100.00"),
			StartingBalance: decimal.RequireFromString("100.00"),
			CreatedAt:       createdAt,
		}
	}
	return rows
}

// -- CreateAccount tests --

func TestCreateAccount_Success(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	amount := decimal.RequireFromString("1000.00")
	expectedID := uuid.Must(uuid.NewV4())
	accountType := accountTypeFromStorage(sqlconfig.AccountTypeCash)

	mockTable.EXPECT().Insert(mock.Anything, mock.MatchedBy(func(c *sqlconfig.AccountCreate) bool {
		return c.Name == "Checking" &&
			c.Type == sqlconfig.AccountTypeCash &&
			c.SubType == "Primary" &&
			c.Balance.Equal(amount) &&
			c.StartingBalance.Equal(amount)
	})).Return(expectedID, nil)

	id, err := svc.CreateAccount(context.Background(), Account{
		Name:            "Checking",
		Type:            accountType,
		SubType:         "Primary",
		Balance:         amount,
		StartingBalance: amount,
	})

	assert.NoError(t, err)
	assert.Equal(t, expectedID, id)
}

func TestCreateAccount_StorageError(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	mockTable.EXPECT().Insert(mock.Anything, mock.Anything).
		Return(uuid.Nil, errors.New("insert failed"))

	id, err := svc.CreateAccount(context.Background(), Account{
		Name:    "Checking",
		SubType: "Primary",
		Balance: decimal.RequireFromString("10.00"),
	})

	assert.Error(t, err)
	assert.Equal(t, "insert failed", err.Error())
	assert.Equal(t, uuid.Nil, id)
}

// -- GetAccount tests --

func TestGetAccount_Success(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	id := uuid.Must(uuid.NewV4())
	createdAt := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	row := &sqlconfig.Account{
		ID:              id,
		Name:            "Checking",
		Type:            sqlconfig.AccountTypeCash,
		SubType:         "Primary",
		Balance:         decimal.RequireFromString("750.00"),
		StartingBalance: decimal.RequireFromString("500.00"),
		CreatedAt:       createdAt,
	}

	mockTable.EXPECT().FindByID(mock.Anything, id).Return(row, nil)

	account, err := svc.GetAccount(context.Background(), id)

	assert.NoError(t, err)
	assert.NotNil(t, account)
	assert.Equal(t, id, account.ID)
	assert.Equal(t, row.Name, account.Name)
	assert.Equal(t, accountTypeFromStorage(row.Type), account.Type)
	assert.Equal(t, row.SubType, account.SubType)
	assert.True(t, row.Balance.Equal(account.Balance))
	assert.True(t, row.StartingBalance.Equal(account.StartingBalance))
	assert.Equal(t, row.CreatedAt, account.CreatedAt)
}

func TestGetAccount_StorageError(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	id := uuid.Must(uuid.NewV4())
	mockTable.EXPECT().FindByID(mock.Anything, id).
		Return(nil, errors.New("account not found"))

	account, err := svc.GetAccount(context.Background(), id)

	assert.Error(t, err)
	assert.Equal(t, "account not found", err.Error())
	assert.Nil(t, account)
}

// -- ListAccounts tests --

func TestListAccounts_NoResults(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	mockTable.EXPECT().List(mock.Anything, mock.Anything).
		Return([]*sqlconfig.Account{}, nil)

	accounts, next, err := svc.ListAccounts(context.Background(), nil)

	assert.NoError(t, err)
	assert.Nil(t, accounts)
	assert.Nil(t, next)
}

func TestListAccounts_SinglePage(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	now := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	rows := makeStorageAccounts(2, now)

	mockTable.EXPECT().List(mock.Anything, mock.MatchedBy(func(f *sqlconfig.AccountFilter) bool {
		return f.Limit == defaultAccountLimit && f.Offset == 0
	})).Return(rows, nil)

	accounts, next, err := svc.ListAccounts(context.Background(), nil)

	assert.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.Nil(t, next)

	account := accounts[0]
	assert.Equal(t, rows[0].ID, account.ID)
	assert.Equal(t, rows[0].Name, account.Name)
	assert.Equal(t, accountTypeFromStorage(rows[0].Type), account.Type)
	assert.Equal(t, rows[0].SubType, account.SubType)
	assert.True(t, rows[0].Balance.Equal(account.Balance))
	assert.True(t, rows[0].StartingBalance.Equal(account.StartingBalance))
	assert.Equal(t, rows[0].CreatedAt, account.CreatedAt)
}

func TestListAccounts_HasNextPage(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	now := time.Date(2025, 7, 1, 12, 0, 0, 0, time.UTC)
	rows := makeStorageAccounts(defaultAccountLimit+1, now)

	mockTable.EXPECT().List(mock.Anything, mock.Anything).Return(rows, nil)

	accounts, next, err := svc.ListAccounts(context.Background(), nil)

	assert.NoError(t, err)
	assert.Len(t, accounts, defaultAccountLimit, "truncated to default account limit")
	assert.NotNil(t, next)
	assert.Equal(t, defaultAccountLimit, next.Position)
	assert.Equal(t, defaultAccountLimit, next.Limit)
}

func TestListAccounts_WithCursor(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	rows := makeStorageAccounts(3, time.Date(2025, 6, 10, 8, 0, 0, 0, time.UTC))

	mockTable.EXPECT().List(mock.Anything, mock.MatchedBy(func(f *sqlconfig.AccountFilter) bool {
		return f.Limit == 2 && f.Offset == 20
	})).Return(rows, nil)

	accounts, next, err := svc.ListAccounts(context.Background(), &AccountCursor{
		Position: 20,
		Limit:    2,
	})

	assert.NoError(t, err)
	assert.Len(t, accounts, 2)
	assert.NotNil(t, next)
	assert.Equal(t, 22, next.Position)
	assert.Equal(t, 2, next.Limit)
}

func TestListAccounts_StorageError(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	mockTable.EXPECT().List(mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))

	accounts, next, err := svc.ListAccounts(context.Background(), nil)

	assert.Error(t, err)
	assert.Equal(t, "database unavailable", err.Error())
	assert.Nil(t, accounts)
	assert.Nil(t, next)
}

// -- UpdateBalance tests --

func TestUpdateBalance_Success(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	id := uuid.Must(uuid.NewV4())
	balance := decimal.RequireFromString("120.25")
	mockTable.EXPECT().UpdateBalance(mock.Anything, id, balance).Return(nil)

	err := svc.UpdateBalance(context.Background(), id, balance)
	assert.NoError(t, err)
}

func TestUpdateBalance_StorageError(t *testing.T) {
	svc, mockTable := newAccountTestService(t)

	id := uuid.Must(uuid.NewV4())
	balance := decimal.RequireFromString("120.25")
	mockTable.EXPECT().UpdateBalance(mock.Anything, id, balance).
		Return(errors.New("update failed"))

	err := svc.UpdateBalance(context.Background(), id, balance)
	assert.Error(t, err)
	assert.Equal(t, "update failed", err.Error())
}
