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

// mockTransactionCreator is a mock for transactionCreator.
type mockTransactionService struct {
	mock.Mock
}

func (m *mockTransactionService) CreateTransaction(ctx context.Context, transaction service.Transaction) (uuid.UUID, error) {
	args := m.Called(ctx, transaction)
	if args.Get(0) == nil {
		return uuid.Nil, args.Error(1)
	}
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// newTestAPI registers the handler against a humatest API and returns it.
func newTestAPI(t *testing.T, svc transactionCreator) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewCreateTransactionHandler(svc).Register(api)
	return api
}

// -- parseCreateTransactionInput unit tests --
// These verify individual parsed field values which the HTTP tests don't assert.

func TestParseCreateTransactionInput_ValidInput(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	transactionDate := "2025-01-15T10:30:00Z"

	input := &CreateTransactionInput{
		Body: CreateTransactionBody{
			AccountID:       accountID.String(),
			CategoryID:      categoryID.String(),
			Amount:          "123.45",
			TransactionName: "Test Transaction",
			TransactionDate: transactionDate,
		},
	}

	parsedAccountID, parsedCategoryID, parsedAmount, parsedName, parsedDate, err := parseCreateTransactionInput(input)
	assert.NoError(t, err)
	assert.Equal(t, accountID, parsedAccountID)
	assert.Equal(t, categoryID, parsedCategoryID)
	assert.True(t, parsedAmount.Equal(decimal.RequireFromString("123.45")))
	assert.Equal(t, "Test Transaction", parsedName)
	expectedDate, _ := time.Parse(time.RFC3339, transactionDate)
	assert.True(t, parsedDate.Equal(expectedDate))
}

func TestParseCreateTransactionInput_ValidInputWithoutDate(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())

	input := &CreateTransactionInput{
		Body: CreateTransactionBody{
			AccountID:       accountID.String(),
			CategoryID:      categoryID.String(),
			Amount:          "-99.99",
			TransactionName: "Refund",
		},
	}

	parsedAccountID, parsedCategoryID, parsedAmount, parsedName, parsedDate, err := parseCreateTransactionInput(input)
	assert.NoError(t, err)
	assert.Equal(t, accountID, parsedAccountID)
	assert.Equal(t, categoryID, parsedCategoryID)
	assert.True(t, parsedAmount.Equal(decimal.RequireFromString("-99.99")))
	assert.Equal(t, "Refund", parsedName)
	assert.True(t, parsedDate.IsZero())
}

// -- HTTP integration tests (full Huma stack via humatest) --

func TestHTTP_CreateTransaction_Success(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txID := uuid.Must(uuid.NewV4())

	mockSvc := new(mockTransactionService)
	mockSvc.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx service.Transaction) bool {
		return tx.AccountID == accountID &&
			tx.CategoryID == categoryID &&
			tx.Amount.Equal(decimal.RequireFromString("12.50")) &&
			tx.TransactionName == "Coffee"
	})).Return(txID, nil)

	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       accountID.String(),
		CategoryID:      categoryID.String(),
		Amount:          "12.50",
		TransactionName: "Coffee",
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	var body CreateTransactionResponse
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, txID.String(), body.ID)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_CreateTransaction_WithDate_Success(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txID := uuid.Must(uuid.NewV4())
	txDate := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	mockSvc := new(mockTransactionService)
	mockSvc.On("CreateTransaction", mock.Anything, mock.MatchedBy(func(tx service.Transaction) bool {
		return tx.TransactionDate.Equal(txDate)
	})).Return(txID, nil)

	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       accountID.String(),
		CategoryID:      categoryID.String(),
		Amount:          "5.00",
		TransactionName: "Lunch",
		TransactionDate: txDate.Format(time.RFC3339),
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockSvc.AssertExpectations(t)
}

func TestHTTP_CreateTransaction_MissingRequiredFields(t *testing.T) {
	mockSvc := new(mockTransactionService)

	// Huma schema validation rejects the request before the handler runs.
	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID: uuid.Must(uuid.NewV4()).String(),
		// CategoryID, Amount, TransactionName omitted
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_TransactionNameTooShort(t *testing.T) {
	mockSvc := new(mockTransactionService)

	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10.00",
		TransactionName: "", // minLength:"1" violation
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_InvalidAccountID(t *testing.T) {
	mockSvc := new(mockTransactionService)

	// Huma's format:"uuid" schema validation rejects this before the handler runs.
	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       "not-a-uuid",
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10.00",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_InvalidCategoryID(t *testing.T) {
	mockSvc := new(mockTransactionService)

	// Huma's format:"uuid" schema validation rejects this before the handler runs.
	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      "not-a-uuid",
		Amount:          "10.00",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_InvalidAmount(t *testing.T) {
	mockSvc := new(mockTransactionService)

	// Amount is a plain string with no Huma format tag, so parseCreateTransactionInput
	// handles validation and returns 400.
	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "not-a-decimal",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_InvalidTransactionDate(t *testing.T) {
	mockSvc := new(mockTransactionService)

	// Huma's format:"date-time" schema validation rejects this before the handler runs.
	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10.00",
		TransactionName: "Test",
		TransactionDate: "not-a-date",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockSvc.AssertNotCalled(t, "CreateTransaction")
}

func TestHTTP_CreateTransaction_ServiceError(t *testing.T) {
	mockSvc := new(mockTransactionService)
	mockSvc.On("CreateTransaction", mock.Anything, mock.Anything).
		Return(uuid.Nil, errors.New("database unavailable"))

	resp := newTestAPI(t, mockSvc).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10.00",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockSvc.AssertExpectations(t)
}
