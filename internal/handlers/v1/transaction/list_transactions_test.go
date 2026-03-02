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

	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

type mockTransactionReader struct {
	mock.Mock
}

func (m *mockTransactionReader) List(ctx context.Context, filter *transaction.TransactionFilter) (*transaction.TransactionListResult, error) {
	args := m.Called(ctx, filter)
	result, _ := args.Get(0).(*transaction.TransactionListResult)
	return result, args.Error(1)
}

func newListTestAPI(t *testing.T, reader transactionReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewListTransactionsHandler(reader).Register(api)
	return api
}

// -- parseListTransactionsInput unit tests --

func TestParseListTransactionsInput_NoCursor(t *testing.T) {
	input := &ListTransactionsInput{
		Body: ListTransactionsBody{},
	}

	filter, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, 20, filter.Limit)
	assert.Equal(t, 0, filter.Offset)
	assert.Nil(t, filter.MaxCreationTime)
}

func TestParseListTransactionsInput_WithCursor(t *testing.T) {
	cursorMaxTime := "2025-06-15T08:00:00Z"

	input := &ListTransactionsInput{
		Body: ListTransactionsBody{
			Cursor: &ListTransactionsCursor{
				Position:        40,
				Limit:           10,
				MaxCreationTime: cursorMaxTime,
			},
		},
	}

	filter, err := parseListTransactionsInput(input)
	assert.NoError(t, err)

	expectedMax, _ := time.Parse(time.RFC3339, cursorMaxTime)
	assert.NotNil(t, filter)
	assert.Equal(t, 40, filter.Offset)
	assert.Equal(t, 10, filter.Limit)
	assert.NotNil(t, filter.MaxCreationTime)
	assert.True(t, filter.MaxCreationTime.Equal(expectedMax))
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

	filter, err := parseListTransactionsInput(input)
	assert.NoError(t, err)
	assert.NotNil(t, filter)
	assert.Equal(t, 0, filter.Offset)
}

// -- HTTP integration tests --

func TestHTTP_ListTransactions_SinglePage(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	txID := uuid.Must(uuid.NewV4())

	mockReader := new(mockTransactionReader)
	mockReader.On("List", mock.Anything, mock.Anything).
		Return(&transaction.TransactionListResult{
			Transactions: []*transaction.Transaction{
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
			NextCursor: nil,
		}, nil)

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Transactions, 1)
	assert.Equal(t, txID.String(), body.Transactions[0].ID)
	assert.Nil(t, body.NextCursor)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListTransactions_MultiplePages(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	svcDefaultLimit := 20

	txs := []*transaction.Transaction{
		{
			ID:              uuid.Must(uuid.NewV4()),
			AccountID:       uuid.Must(uuid.NewV4()),
			CategoryID:      uuid.Must(uuid.NewV4()),
			Amount:          decimal.RequireFromString("5.00"),
			TransactionName: "Item",
			TransactionDate: now,
			CreatedAt:       now,
		},
		{
			ID:              uuid.Must(uuid.NewV4()),
			AccountID:       uuid.Must(uuid.NewV4()),
			CategoryID:      uuid.Must(uuid.NewV4()),
			Amount:          decimal.RequireFromString("5.00"),
			TransactionName: "Item",
			TransactionDate: now,
			CreatedAt:       now,
		},
	}

	mockReader := new(mockTransactionReader)
	mockReader.On("List", mock.Anything, mock.Anything).
		Return(&transaction.TransactionListResult{
			Transactions: txs,
			NextCursor: &transaction.TransactionCursor{
				Position:        svcDefaultLimit,
				Limit:           svcDefaultLimit,
				MaxCreationTime: now,
			},
		}, nil)

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Transactions, 2)
	assert.NotNil(t, body.NextCursor)
	assert.Equal(t, svcDefaultLimit, body.NextCursor.Position)
	assert.Equal(t, svcDefaultLimit, body.NextCursor.Limit)
	assert.Equal(t, now.Format(time.RFC3339), body.NextCursor.MaxCreationTime)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListTransactions_WithCursor(t *testing.T) {
	maxTime := time.Date(2025, 5, 1, 10, 0, 0, 0, time.UTC)

	mockReader := new(mockTransactionReader)
	mockReader.On("List", mock.Anything, mock.MatchedBy(func(f *transaction.TransactionFilter) bool {
		return f != nil &&
			f.Offset == 40 &&
			f.Limit == 10 &&
			f.MaxCreationTime != nil &&
			f.MaxCreationTime.Equal(maxTime)
	})).Return(&transaction.TransactionListResult{
		Transactions: nil,
		NextCursor:   nil,
	}, nil)

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{
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
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListTransactions_NoResults(t *testing.T) {
	mockReader := new(mockTransactionReader)
	mockReader.On("List", mock.Anything, mock.Anything).
		Return(&transaction.TransactionListResult{
			Transactions: nil,
			NextCursor:   nil,
		}, nil)

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusOK, resp.Code)
	var body ListTransactionsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Empty(t, body.Transactions)
	assert.Nil(t, body.NextCursor)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListTransactions_ServiceError(t *testing.T) {
	mockReader := new(mockTransactionReader)
	mockReader.On("List", mock.Anything, mock.Anything).
		Return((*transaction.TransactionListResult)(nil), errors.New("database unavailable"))

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListTransactions_InvalidCursorMaxCreationTime(t *testing.T) {
	mockReader := new(mockTransactionReader)

	resp := newListTestAPI(t, mockReader).Post("/v1/transaction/list", ListTransactionsBody{
		Cursor: &ListTransactionsCursor{
			Position:        0,
			Limit:           10,
			MaxCreationTime: "not-a-date",
		},
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockReader.AssertNotCalled(t, "List")
}
