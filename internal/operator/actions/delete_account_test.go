package actions

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

func TestDeleteAccount_Perform_Success(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	existing := &account.Account{
		ID:   accID,
		Name: "Checking",
		Type: account.AccountTypeCash,
	}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Delete(mock.Anything, accID).
		Return(nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		DeleteByAccountID(mock.Anything, accID).
		Return(nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		DeleteAccountLinksByAccountID(mock.Anything, accID).
		Return(nil)

	mockSync := &storage.MockISyncWriter{}
	mockSync.EXPECT().
		Delete(mock.Anything, accID).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid
	wt.Sync = mockSync

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	require.NoError(t, err)
	mockAccount.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
	mockPlaid.AssertExpectations(t)
	mockSync.AssertExpectations(t)
}

func TestDeleteAccount_Perform_AccountNotFound(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	mockAccount.AssertExpectations(t)
	mockAccount.AssertNotCalled(t, "Delete")
}

func TestDeleteAccount_Perform_NilExisting(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(nil, nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAccountNotFound)
	mockAccount.AssertExpectations(t)
}

func TestDeleteAccount_Perform_DeleteTransactionsError(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	txnErr := errors.New("txn delete failed")
	existing := &account.Account{ID: accID, Name: "Checking", Type: account.AccountTypeCash}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		DeleteByAccountID(mock.Anything, accID).
		Return(txnErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	wt.Transaction = mockTxn

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	assert.ErrorIs(t, err, txnErr)
	mockAccount.AssertNotCalled(t, "Delete")
}

func TestDeleteAccount_Perform_DeleteAccountLinksError(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	linkErr := errors.New("link delete failed")
	existing := &account.Account{ID: accID, Name: "Checking", Type: account.AccountTypeCash}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		DeleteByAccountID(mock.Anything, accID).
		Return(nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		DeleteAccountLinksByAccountID(mock.Anything, accID).
		Return(linkErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	assert.ErrorIs(t, err, linkErr)
	mockAccount.AssertNotCalled(t, "Delete")
}

func TestDeleteAccount_Perform_DeleteSyncError(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	syncErr := errors.New("sync delete failed")
	existing := &account.Account{ID: accID, Name: "Checking", Type: account.AccountTypeCash}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		DeleteByAccountID(mock.Anything, accID).
		Return(nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		DeleteAccountLinksByAccountID(mock.Anything, accID).
		Return(nil)

	mockSync := &storage.MockISyncWriter{}
	mockSync.EXPECT().
		Delete(mock.Anything, accID).
		Return(syncErr)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid
	wt.Sync = mockSync

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	assert.ErrorIs(t, err, syncErr)
	mockAccount.AssertNotCalled(t, "Delete")
}

func TestDeleteAccount_Perform_DeleteAccountError(t *testing.T) {
	accID := uuid.Must(uuid.NewV4())
	deleteErr := errors.New("account delete failed")
	existing := &account.Account{ID: accID, Name: "Checking", Type: account.AccountTypeCash}

	mockAccount := &storage.MockIAccountWriter{}
	mockAccount.EXPECT().
		FindByIDForUpdate(mock.Anything, accID).
		Return(existing, nil)
	mockAccount.EXPECT().
		Delete(mock.Anything, accID).
		Return(deleteErr)

	mockTxn := &storage.MockITransactionWriter{}
	mockTxn.EXPECT().
		DeleteByAccountID(mock.Anything, accID).
		Return(nil)

	mockPlaid := &storage.MockIPlaidWriter{}
	mockPlaid.EXPECT().
		DeleteAccountLinksByAccountID(mock.Anything, accID).
		Return(nil)

	mockSync := &storage.MockISyncWriter{}
	mockSync.EXPECT().
		Delete(mock.Anything, accID).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Account = mockAccount
	wt.Transaction = mockTxn
	wt.Plaid = mockPlaid
	wt.Sync = mockSync

	err := (&DeleteAccount{ID: accID}).Perform(context.Background(), wt)
	assert.ErrorIs(t, err, deleteErr)
}
