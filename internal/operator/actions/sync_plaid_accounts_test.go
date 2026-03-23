package actions

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/account"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

// mustID is a test helper to generate a UUID.
func mustID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV4()
	require.NoError(t, err)
	return id
}

func txnDate(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

// newSyncWriter returns a test Writer with fresh mocks for account, transaction, and plaid.
func newSyncWriter(t *testing.T) (*storage.Writer, *storage.MockIAccountWriter, *storage.MockITransactionWriter, *storage.MockIPlaidWriter) {
	t.Helper()
	mockAccount := &storage.MockIAccountWriter{}
	mockTxn := &storage.MockITransactionWriter{}
	mockPlaid := &storage.MockIPlaidWriter{}
	w := storage.NewWriterForTest()
	w.Account = mockAccount
	w.Transaction = mockTxn
	w.Plaid = mockPlaid
	return w, mockAccount, mockTxn, mockPlaid
}

// --- tests ---

func TestSyncPlaidAccounts_Perform_NewTransaction(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	txnID := mustID(t)
	plaidAccID := "plaid-acc"
	link := &plaidstore.AccountLink{PlaidAccountID: plaidAccID, AccountID: accID, PlaidItemID: itemID}
	amount := decimal.NewFromFloat(-12.50)
	date := txnDate("2025-03-01")

	w, mockAccount, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{link}, nil)
	mockPlaid.On("FindTransactionLink", mock.Anything, "plaid-txn-1").
		Return((*plaidstore.TransactionLink)(nil), nil)
	mockTxn.On("Insert", mock.Anything, &transaction.TransactionCreate{
		AccountID:       accID,
		CategoryID:      nil,
		Amount:          amount,
		TransactionName: "Coffee",
		TransactionDate: date,
	}).Return(txnID, nil)
	mockPlaid.On("CreateTransactionLink", mock.Anything, &plaidstore.TransactionLink{
		PlaidTransactionID: "plaid-txn-1",
		TransactionID:      txnID,
		PlaidAccountID:     plaidAccID,
	}).Return(nil)
	mockAccount.On("FindByIDForUpdate", mock.Anything, accID).
		Return(&account.Account{ID: accID, Balance: decimal.NewFromInt(100)}, nil)
	mockAccount.On("UpdateBalance", mock.Anything, accID, decimal.NewFromFloat(87.5)).
		Return(nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "cursor-next").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{{
			ItemID: itemID,
			Transactions: []SyncedTransaction{
				{PlaidTransactionID: "plaid-txn-1", PlaidAccountID: plaidAccID, Amount: amount, Name: "Coffee", Date: date},
			},
			NextCursor: "cursor-next",
		}},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestSyncPlaidAccounts_Perform_RemoveTransaction(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	internalTxnID := mustID(t)
	plaidAccID := "plaid-acc"

	accountLink := &plaidstore.AccountLink{PlaidAccountID: plaidAccID, AccountID: accID, PlaidItemID: itemID}
	txnLink := &plaidstore.TransactionLink{PlaidTransactionID: "plaid-txn-del", TransactionID: internalTxnID, PlaidAccountID: plaidAccID}
	deletedTxn := &transaction.Transaction{ID: internalTxnID, AccountID: accID, Amount: decimal.NewFromFloat(-25.00)}

	w, mockAccount, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{accountLink}, nil)
	mockPlaid.On("FindTransactionLink", mock.Anything, "plaid-txn-del").
		Return(txnLink, nil)
	mockTxn.On("DeleteByID", mock.Anything, internalTxnID).
		Return(deletedTxn, nil)
	mockPlaid.On("DeleteTransactionLink", mock.Anything, "plaid-txn-del").
		Return(nil)
	mockAccount.On("FindByIDForUpdate", mock.Anything, accID).
		Return(&account.Account{ID: accID, Balance: decimal.NewFromFloat(75.00)}, nil)
	// balance reversal: 75 - (-25) = 100
	mockAccount.On("UpdateBalance", mock.Anything, accID, decimal.NewFromFloat(75.00).Sub(decimal.NewFromFloat(-25.00))).
		Return(nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "cursor-2").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{{
			ItemID: itemID,
			Transactions: []SyncedTransaction{
				{PlaidTransactionID: "plaid-txn-del", PlaidAccountID: plaidAccID, IsRemoved: true},
			},
			NextCursor: "cursor-2",
		}},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestSyncPlaidAccounts_Perform_ModifyTransaction(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	oldTxnID := mustID(t)
	plaidAccID := "plaid-acc"

	accountLink := &plaidstore.AccountLink{PlaidAccountID: plaidAccID, AccountID: accID, PlaidItemID: itemID}
	txnLink := &plaidstore.TransactionLink{PlaidTransactionID: "plaid-txn-mod", TransactionID: oldTxnID, PlaidAccountID: plaidAccID}
	oldTxn := &transaction.Transaction{ID: oldTxnID, AccountID: accID, Amount: decimal.NewFromFloat(-10.00)}
	newAmount := decimal.NewFromFloat(-15.00)
	date := txnDate("2025-04-01")

	w, mockAccount, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{accountLink}, nil)
	mockPlaid.On("FindTransactionLink", mock.Anything, "plaid-txn-mod").
		Return(txnLink, nil)
	mockTxn.On("FindByID", mock.Anything, oldTxnID).Return(oldTxn, nil)
	mockTxn.On("Update", mock.Anything, oldTxnID, &transaction.TransactionUpdate{
		Amount:          newAmount,
		TransactionName: "Dinner",
		TransactionDate: date,
	}).Return(nil)
	// delta: -15 - (-10) = -5; balance: 100 + (-5) = 95
	mockAccount.On("FindByIDForUpdate", mock.Anything, accID).
		Return(&account.Account{ID: accID, Balance: decimal.NewFromInt(100)}, nil)
	mockAccount.On("UpdateBalance", mock.Anything, accID, decimal.NewFromFloat(95.00)).
		Return(nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "cursor-3").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{{
			ItemID: itemID,
			Transactions: []SyncedTransaction{
				{PlaidTransactionID: "plaid-txn-mod", PlaidAccountID: plaidAccID, Amount: newAmount, Name: "Dinner", Date: date},
			},
			NextCursor: "cursor-3",
		}},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}

func TestSyncPlaidAccounts_Perform_UnknownPlaidAccount_Skipped(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)

	// Link covers "known-acc"; transaction references "unknown-acc".
	knownLink := &plaidstore.AccountLink{PlaidAccountID: "known-acc", AccountID: accID, PlaidItemID: itemID}

	w, _, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{knownLink}, nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "c").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{{
			ItemID: itemID,
			Transactions: []SyncedTransaction{
				{PlaidTransactionID: "txn-x", PlaidAccountID: "unknown-acc", Amount: decimal.NewFromFloat(-5.0)},
			},
			NextCursor: "c",
		}},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockTxn.AssertNotCalled(t, "Insert")
	w.Account.(*storage.MockIAccountWriter).AssertNotCalled(t, "UpdateBalance")
}

func TestSyncPlaidAccounts_Perform_ListAccountLinksError(t *testing.T) {
	itemID := mustID(t)

	w, _, _, mockPlaid := newSyncWriter(t)
	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return(([]*plaidstore.AccountLink)(nil), errors.New("db error"))

	action := &SyncPlaidAccounts{Items: []SyncedItem{{ItemID: itemID, NextCursor: "c"}}}
	err := action.Perform(context.Background(), w)
	assert.Error(t, err)
}

func TestSyncPlaidAccounts_Perform_RemoveNonexistentLink_IsNoOp(t *testing.T) {
	itemID := mustID(t)
	accID := mustID(t)
	plaidAccID := "p-acc"
	link := &plaidstore.AccountLink{PlaidAccountID: plaidAccID, AccountID: accID, PlaidItemID: itemID}

	w, _, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{link}, nil)
	// No link exists for this transaction — should be silently skipped.
	mockPlaid.On("FindTransactionLink", mock.Anything, "ghost-txn").
		Return((*plaidstore.TransactionLink)(nil), nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "c").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{{
			ItemID: itemID,
			Transactions: []SyncedTransaction{
				{PlaidTransactionID: "ghost-txn", PlaidAccountID: plaidAccID, IsRemoved: true},
			},
			NextCursor: "c",
		}},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockTxn.AssertNotCalled(t, "DeleteByID")
}

func TestSyncPlaidAccounts_Perform_UpdateCursorError(t *testing.T) {
	itemID := mustID(t)

	w, _, _, mockPlaid := newSyncWriter(t)
	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, itemID).
		Return([]*plaidstore.AccountLink{}, nil)
	mockPlaid.On("UpdateCursor", mock.Anything, itemID, "c").Return(errors.New("cursor update failed"))

	action := &SyncPlaidAccounts{Items: []SyncedItem{{ItemID: itemID, NextCursor: "c"}}}
	err := action.Perform(context.Background(), w)
	assert.Error(t, err)
}

func TestSyncPlaidAccounts_Perform_MultipleItems_SingleTransaction(t *testing.T) {
	item1ID := mustID(t)
	item2ID := mustID(t)
	acc1ID := mustID(t)
	acc2ID := mustID(t)
	txn1ID := mustID(t)
	txn2ID := mustID(t)

	link1 := &plaidstore.AccountLink{PlaidAccountID: "p-acc-1", AccountID: acc1ID, PlaidItemID: item1ID}
	link2 := &plaidstore.AccountLink{PlaidAccountID: "p-acc-2", AccountID: acc2ID, PlaidItemID: item2ID}
	amount := decimal.NewFromFloat(-20.00)
	date := txnDate("2025-05-01")

	w, mockAccount, mockTxn, mockPlaid := newSyncWriter(t)

	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, item1ID).Return([]*plaidstore.AccountLink{link1}, nil)
	mockPlaid.On("ListAccountLinksByItemID", mock.Anything, item2ID).Return([]*plaidstore.AccountLink{link2}, nil)

	mockPlaid.On("FindTransactionLink", mock.Anything, "txn-a").Return((*plaidstore.TransactionLink)(nil), nil)
	mockPlaid.On("FindTransactionLink", mock.Anything, "txn-b").Return((*plaidstore.TransactionLink)(nil), nil)

	mockTxn.On("Insert", mock.Anything, &transaction.TransactionCreate{AccountID: acc1ID, Amount: amount, TransactionName: "A", TransactionDate: date}).Return(txn1ID, nil)
	mockTxn.On("Insert", mock.Anything, &transaction.TransactionCreate{AccountID: acc2ID, Amount: amount, TransactionName: "B", TransactionDate: date}).Return(txn2ID, nil)

	mockPlaid.On("CreateTransactionLink", mock.Anything, &plaidstore.TransactionLink{PlaidTransactionID: "txn-a", TransactionID: txn1ID, PlaidAccountID: "p-acc-1"}).Return(nil)
	mockPlaid.On("CreateTransactionLink", mock.Anything, &plaidstore.TransactionLink{PlaidTransactionID: "txn-b", TransactionID: txn2ID, PlaidAccountID: "p-acc-2"}).Return(nil)

	mockAccount.On("FindByIDForUpdate", mock.Anything, acc1ID).Return(&account.Account{ID: acc1ID, Balance: decimal.NewFromInt(500)}, nil)
	mockAccount.On("FindByIDForUpdate", mock.Anything, acc2ID).Return(&account.Account{ID: acc2ID, Balance: decimal.NewFromInt(200)}, nil)
	mockAccount.On("UpdateBalance", mock.Anything, acc1ID, decimal.NewFromInt(500).Add(amount)).Return(nil)
	mockAccount.On("UpdateBalance", mock.Anything, acc2ID, decimal.NewFromInt(200).Add(amount)).Return(nil)

	mockPlaid.On("UpdateCursor", mock.Anything, item1ID, "c1").Return(nil)
	mockPlaid.On("UpdateCursor", mock.Anything, item2ID, "c2").Return(nil)

	action := &SyncPlaidAccounts{
		Items: []SyncedItem{
			{ItemID: item1ID, Transactions: []SyncedTransaction{{PlaidTransactionID: "txn-a", PlaidAccountID: "p-acc-1", Amount: amount, Name: "A", Date: date}}, NextCursor: "c1"},
			{ItemID: item2ID, Transactions: []SyncedTransaction{{PlaidTransactionID: "txn-b", PlaidAccountID: "p-acc-2", Amount: amount, Name: "B", Date: date}}, NextCursor: "c2"},
		},
	}

	err := action.Perform(context.Background(), w)
	require.NoError(t, err)
	mockPlaid.AssertExpectations(t)
	mockTxn.AssertExpectations(t)
	mockAccount.AssertExpectations(t)
}
