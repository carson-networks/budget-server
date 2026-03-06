package transaction

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
)

func newCreateTransactionTestAPI(t *testing.T, op operator.IProcessor) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewCreateTransactionHandler(op).Register(api)
	return api
}

func TestHTTP_CreateTransaction_SuccessWithExplicitDate(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	txnDate := time.Date(2025, 3, 5, 12, 0, 0, 0, time.UTC)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			ct, ok := a.(*actions.CreateTransaction)
			return ok &&
				ct.AccountID == accountID &&
				ct.CategoryID == categoryID &&
				ct.Amount.Equal(decimal.NewFromInt(-50)) &&
				ct.TransactionName == "Groceries" &&
				ct.TransactionDate.Equal(txnDate)
		})).
		Return(nil)

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       accountID.String(),
		CategoryID:      categoryID.String(),
		Amount:          "-50",
		TransactionName: "Groceries",
		TransactionDate: txnDate.Format(time.RFC3339),
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateTransaction_InvalidAccountID(t *testing.T) {
	mockOp := &operator.MockIProcessor{}

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       "not-a-uuid",
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}

func TestHTTP_CreateTransaction_InvalidCategoryID(t *testing.T) {
	mockOp := &operator.MockIProcessor{}

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      "not-a-uuid",
		Amount:          "10",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}

func TestHTTP_CreateTransaction_InvalidAmount(t *testing.T) {
	mockOp := &operator.MockIProcessor{}

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "not-a-number",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}

func TestHTTP_CreateTransaction_InvalidTransactionDate(t *testing.T) {
	mockOp := &operator.MockIProcessor{}

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10",
		TransactionName: "Test",
		TransactionDate: "not-a-date",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}

func TestHTTP_CreateTransaction_ProcessReturnsError(t *testing.T) {
	processErr := errors.New("database unavailable")
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(processErr)

	resp := newCreateTransactionTestAPI(t, mockOp).Post("/v1/transaction", CreateTransactionBody{
		AccountID:       uuid.Must(uuid.NewV4()).String(),
		CategoryID:      uuid.Must(uuid.NewV4()).String(),
		Amount:          "10",
		TransactionName: "Test",
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockOp.AssertExpectations(t)
}
