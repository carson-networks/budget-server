package account

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	plaidstore "github.com/carson-networks/budget-server/internal/storage/plaid"
)

// ---- mock: syncPlaidReader ----

type mockSyncPlaidReader struct {
	mock.Mock
}

func (m *mockSyncPlaidReader) ListItemsForSync(ctx context.Context, accountIDs []uuid.UUID) ([]*plaidstore.PlaidItem, error) {
	args := m.Called(ctx, accountIDs)
	items, _ := args.Get(0).([]*plaidstore.PlaidItem)
	return items, args.Error(1)
}

// ---- mock: plaidSyncer ----

type mockPlaidSyncer struct {
	mock.Mock
}

func (m *mockPlaidSyncer) SyncTransactions(ctx context.Context, accessToken, cursor string) (*plaidclient.SyncResult, error) {
	args := m.Called(ctx, accessToken, cursor)
	result, _ := args.Get(0).(*plaidclient.SyncResult)
	return result, args.Error(1)
}

// ---- test helpers ----

func newSyncTestAPI(t *testing.T, op operator.IProcessor, syncer plaidSyncer, reader syncPlaidReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	h := &SyncAccountsHandler{Operator: op, PlaidClient: syncer, PlaidReader: reader}
	h.Register(api)
	return api
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV4()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// ---- tests ----

func TestHTTP_SyncAccounts_NoLinkedAccounts_ReturnsEmpty(t *testing.T) {
	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{}, nil)

	resp := newSyncTestAPI(t, nil, nil, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 0, body.Body.SyncedAccounts)
	assert.Equal(t, 0, body.Body.AddedTransactions)
	assert.Equal(t, 0, body.Body.RemovedTransactions)
	reader.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_SyncAll_Success(t *testing.T) {
	itemID := mustUUID(t)
	plaidAccID := "plaid-acc-1"

	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "access-token", Cursor: "cursor-1"}

	syncResult := &plaidclient.SyncResult{
		Added: []plaidclient.SyncTransaction{
			{PlaidTransactionID: "txn-1", PlaidAccountID: plaidAccID, Amount: 10.00, Name: "Coffee", Date: "2025-01-15"},
			{PlaidTransactionID: "txn-2", PlaidAccountID: plaidAccID, Amount: 5.00, Name: "Bus", Date: "2025-01-16"},
		},
		Removed:    []string{"txn-old"},
		NextCursor: "cursor-2",
		HasMore:    false,
	}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "access-token", "cursor-1").Return(syncResult, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			sa, ok := a.(*actions.SyncPlaidAccounts)
			return ok && len(sa.Items) == 1 && sa.Items[0].ItemID == itemID && sa.Items[0].NextCursor == "cursor-2"
		})).
		Return(nil)

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 1, body.Body.SyncedAccounts)
	assert.Equal(t, 2, body.Body.AddedTransactions)
	assert.Equal(t, 1, body.Body.RemovedTransactions)
	reader.AssertExpectations(t)
	syncer.AssertExpectations(t)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_ByAccountIDs_Success(t *testing.T) {
	itemID := mustUUID(t)
	accID := mustUUID(t)
	plaidAccID := "plaid-acc-2"

	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at-2", Cursor: "c-1"}
	_ = plaidAccID

	syncResult := &plaidclient.SyncResult{
		Added:      []plaidclient.SyncTransaction{{PlaidTransactionID: "t1", PlaidAccountID: plaidAccID, Amount: 20.0, Name: "Lunch", Date: "2025-02-01"}},
		NextCursor: "c-2",
		HasMore:    false,
	}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{accID}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at-2", "c-1").Return(syncResult, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().Process(mock.Anything, mock.Anything).Return(nil)

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{
		AccountIDs: []string{accID.String()},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 1, body.Body.SyncedAccounts)
	assert.Equal(t, 1, body.Body.AddedTransactions)
	assert.Equal(t, 0, body.Body.RemovedTransactions)
	reader.AssertExpectations(t)
	syncer.AssertExpectations(t)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_InvalidAccountID_Returns400(t *testing.T) {
	reader := &mockSyncPlaidReader{}

	resp := newSyncTestAPI(t, nil, nil, reader).Post("/v1/accounts/sync", SyncAccountsBody{
		AccountIDs: []string{"not-a-uuid"},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	reader.AssertNotCalled(t, "ListItemsForSync")
}

func TestHTTP_SyncAccounts_ListItemsError_Returns500(t *testing.T) {
	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return(nil, errors.New("db error"))

	resp := newSyncTestAPI(t, nil, nil, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	reader.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_ListAccountLinksError_Returns500(t *testing.T) {
	accID := mustUUID(t)

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{accID}).Return(nil, errors.New("db error"))

	resp := newSyncTestAPI(t, nil, nil, reader).Post("/v1/accounts/sync", SyncAccountsBody{
		AccountIDs: []string{accID.String()},
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	reader.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_PlaidSyncError_Returns500(t *testing.T) {
	itemID := mustUUID(t)
	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at", Cursor: "c"}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at", "c").Return(nil, errors.New("plaid unavailable"))

	resp := newSyncTestAPI(t, nil, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	reader.AssertExpectations(t)
	syncer.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_OperatorError_Returns500(t *testing.T) {
	itemID := mustUUID(t)
	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at", Cursor: "c"}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at", "c").Return(&plaidclient.SyncResult{HasMore: false, NextCursor: "c2"}, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().Process(mock.Anything, mock.Anything).Return(errors.New("operator error"))

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_Pagination_CollectsAllPages(t *testing.T) {
	itemID := mustUUID(t)
	plaidAccID := "p-acc"
	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at", Cursor: ""}

	page1 := &plaidclient.SyncResult{
		Added:      []plaidclient.SyncTransaction{{PlaidTransactionID: "t1", PlaidAccountID: plaidAccID, Amount: 1.0, Name: "A", Date: "2025-01-01"}},
		NextCursor: "cursor-mid",
		HasMore:    true,
	}
	page2 := &plaidclient.SyncResult{
		Added:      []plaidclient.SyncTransaction{{PlaidTransactionID: "t2", PlaidAccountID: plaidAccID, Amount: 2.0, Name: "B", Date: "2025-01-02"}},
		NextCursor: "cursor-final",
		HasMore:    false,
	}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at", "").Return(page1, nil)
	syncer.On("SyncTransactions", mock.Anything, "at", "cursor-mid").Return(page2, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			sa, ok := a.(*actions.SyncPlaidAccounts)
			return ok && len(sa.Items) == 1 && sa.Items[0].NextCursor == "cursor-final" && len(sa.Items[0].Transactions) == 2
		})).
		Return(nil)

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 2, body.Body.AddedTransactions)
	reader.AssertExpectations(t)
	syncer.AssertExpectations(t)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_PendingTransactions_AreSkipped(t *testing.T) {
	itemID := mustUUID(t)
	plaidAccID := "p-acc"
	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at", Cursor: "c"}

	syncResult := &plaidclient.SyncResult{
		Added: []plaidclient.SyncTransaction{
			{PlaidTransactionID: "settled", PlaidAccountID: plaidAccID, Amount: 10.0, Name: "Settled", Date: "2025-01-01", Pending: false},
			{PlaidTransactionID: "pending", PlaidAccountID: plaidAccID, Amount: 5.0, Name: "Pending", Date: "2025-01-01", Pending: true},
		},
		NextCursor: "c2",
		HasMore:    false,
	}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{}).Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at", "c").Return(syncResult, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			sa, ok := a.(*actions.SyncPlaidAccounts)
			return ok && len(sa.Items) == 1 && len(sa.Items[0].Transactions) == 1 && sa.Items[0].Transactions[0].PlaidTransactionID == "settled"
		})).
		Return(nil)

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 1, body.Body.AddedTransactions)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SyncAccounts_DuplicateItemIDs_DeduplicatedAcrossAccounts(t *testing.T) {
	itemID := mustUUID(t)
	acc1ID := mustUUID(t)
	acc2ID := mustUUID(t)

	// Two internal accounts belonging to the same Plaid Item — reader returns it once (DISTINCT in SQL).
	item := &plaidstore.PlaidItem{ID: itemID, AccessToken: "at", Cursor: "c"}

	reader := &mockSyncPlaidReader{}
	reader.On("ListItemsForSync", mock.Anything, []uuid.UUID{acc1ID, acc2ID}).
		Return([]*plaidstore.PlaidItem{item}, nil)

	syncer := &mockPlaidSyncer{}
	syncer.On("SyncTransactions", mock.Anything, "at", "c").
		Return(&plaidclient.SyncResult{NextCursor: "c2", HasMore: false}, nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().Process(mock.Anything, mock.Anything).Return(nil)

	resp := newSyncTestAPI(t, mockOp, syncer, reader).Post("/v1/accounts/sync", SyncAccountsBody{
		AccountIDs: []string{acc1ID.String(), acc2ID.String()},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body SyncAccountsOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, 1, body.Body.SyncedAccounts) // 1 item, not 2 accounts
	reader.AssertExpectations(t)
	mockOp.AssertExpectations(t)
}
