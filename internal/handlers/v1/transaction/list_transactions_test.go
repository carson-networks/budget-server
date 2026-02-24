package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/service"
)

type mockTransactionLister struct {
	mock.Mock
}

func (m *mockTransactionLister) ListTransactions(ctx context.Context, query service.TransactionListQuery) (*service.TransactionListResult, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.TransactionListResult), args.Error(1)
}

func newListTestAPI(t *testing.T, svc transactionLister) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewListTransactionsHandler(svc).Register(api)
	return api
}

// -- parseListTransactionsInput unit tests --

func TestParseListTransactionsInput_LimitOnly(t *testing.T) {
	input := &ListTransactionsInput{
		Body: ListTransactionsBody{
			Limit: 50,
		},
	}

	query, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.Equal(t, 50, query.Limit)
	assert.Nil(t, query.Cursor)
}

func TestParseListTransactionsInput_WithCursor(t *testing.T) {
	cursorMaxTime := "2025-06-15T08:00:00Z"

	input := &ListTransactionsInput{
		Body: ListTransactionsBody{
			Limit: 99,
			Cursor: &ListTransactionsCursor{
				Position:        40,
				Limit:           10,
				MaxCreationTime: cursorMaxTime,
			},
		},
	}

	query, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.Equal(t, 10, query.Limit, "cursor limit overrides top-level limit")

	expectedMax, _ := time.Parse(time.RFC3339, cursorMaxTime)
	assert.NotNil(t, query.Cursor)
	assert.Equal(t, 40, query.Cursor.Position)
	assert.Equal(t, 10, query.Cursor.Limit)
	assert.Equal(t, expectedMax, query.Cursor.MaxCreationTime)
}

func TestParseListTransactionsInput_NoOptionalFields(t *testing.T) {
	input := &ListTransactionsInput{
		Body: ListTransactionsBody{},
	}

	query, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.Equal(t, defaultLimit, query.Limit)
	assert.Nil(t, query.Cursor)
}

func TestParseListTransactionsInput_InvalidCursorMaxCreationTime(t *testing.T) {
	input := &ListTransactionsInput{
		Body: ListTransactionsBody{
			Cursor: &ListTransactionsCursor{
				Position:        0,
				Limit:           10,
				MaxCreationTime: "not-a-date",
			},
		},
	}

	_, err := parseListTransactionsInput(input)
	assert.Error(t, err)
}

func TestParseListTransactionsInput_CursorPositionZero(t *testing.T) {
	input := &ListTransactionsInput{
		Body: ListTransactionsBody{
			Cursor: &ListTransactionsCursor{
				Position:        0,
				Limit:           20,
				MaxCreationTime: "2025-06-01T12:00:00Z",
			},
		},
	}

	query, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.NotNil(t, query.Cursor)
	assert.Equal(t, 0, query.Cursor.Position)
}

// -- HTTP integration tests --

func TestHTTP_ListTransactions_SinglePage(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	txID := uuid.Must(uuid.NewV4())

	mockSvc := new(mockTransactionLister)
	mockSvc.On("ListTransactions", mock.Anything, mock.MatchedBy(func(q service.TransactionListQuery) bool {
		return q.Limit == defaultLimit && q.Cursor == nil
	})).Return(&service.TransactionListResult{
		Transactions: []service.Transaction{
			{
				ID:              txID,
				AccountID:       uuid.Must(uuid.NewV4()),
				CategoryID:      uuid.Must(uuid.NewV4()),
				Amount:          decimal.RequireFromString("10.00"),
				TransactionName: "Coffee",
				TransactionDate: now,
				CreatedAt:       now,
			},
		},
	}, nil)

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Transactions, 1)
	assert.Equal(t, txID.String(), body.Transactions[0].ID)
	assert.Nil(t, body.NextCursor)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_ListTransactions_MultiplePages(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	txs := make([]service.Transaction, 2)
	for i := range txs {
		txs[i] = service.Transaction{
			ID:              uuid.Must(uuid.NewV4()),
			AccountID:       uuid.Must(uuid.NewV4()),
			CategoryID:      uuid.Must(uuid.NewV4()),
			Amount:          decimal.RequireFromString("5.00"),
			TransactionName: "Item",
			TransactionDate: now,
			CreatedAt:       now,
		}
	}

	mockSvc := new(mockTransactionLister)
	mockSvc.On("ListTransactions", mock.Anything, mock.MatchedBy(func(q service.TransactionListQuery) bool {
		return q.Limit == 2
	})).Return(&service.TransactionListResult{
		Transactions: txs,
		NextCursor: &service.TransactionCursor{
			Position:        2,
			Limit:           2,
			MaxCreationTime: now,
		},
	}, nil)

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{
		Limit: 2,
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Transactions, 2)
	assert.NotNil(t, body.NextCursor)
	assert.Equal(t, 2, body.NextCursor.Position)
	assert.Equal(t, 2, body.NextCursor.Limit)
	assert.Equal(t, now.Format(time.RFC3339), body.NextCursor.MaxCreationTime)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_ListTransactions_WithCursor(t *testing.T) {
	maxTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)

	mockSvc := new(mockTransactionLister)
	mockSvc.On("ListTransactions", mock.Anything, mock.MatchedBy(func(q service.TransactionListQuery) bool {
		return q.Cursor != nil &&
			q.Cursor.Position == 40 &&
			q.Limit == 10 &&
			q.Cursor.MaxCreationTime.Equal(maxTime)
	})).Return(&service.TransactionListResult{}, nil)

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{
		Cursor: &ListTransactionsCursor{
			Position:        40,
			Limit:           10,
			MaxCreationTime: maxTime.Format(time.RFC3339),
		},
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Empty(t, body.Transactions)
	assert.Nil(t, body.NextCursor)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_ListTransactions_NoResults(t *testing.T) {
	mockSvc := new(mockTransactionLister)
	mockSvc.On("ListTransactions", mock.Anything, mock.Anything).
		Return(&service.TransactionListResult{}, nil)

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Empty(t, body.Transactions)
	assert.Nil(t, body.NextCursor)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_ListTransactions_ServiceError(t *testing.T) {
	mockSvc := new(mockTransactionLister)
	mockSvc.On("ListTransactions", mock.Anything, mock.Anything).
		Return(nil, errors.New("database unavailable"))

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_ListTransactions_InvalidCursorMaxCreationTime(t *testing.T) {
	mockSvc := new(mockTransactionLister)

	resp := newListTestAPI(t, mockSvc).Post("/v1/transaction/list", ListTransactionsBody{
		Cursor: &ListTransactionsCursor{
			Position:        0,
			Limit:           10,
			MaxCreationTime: "not-a-date",
		},
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "ListTransactions")
}
