package transaction

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/storage/transaction"
)

type mockTransactionTotalsReader struct {
	mock.Mock
}

func (m *mockTransactionTotalsReader) TotalsByMonthAndCategory(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]transaction.MonthTotals, error) {
	args := m.Called(ctx, startMonth, startYear, endMonth, endYear)
	slice, _ := args.Get(0).([]transaction.MonthTotals)
	return slice, args.Error(1)
}

func newTotalsTestAPI(t *testing.T, reader transactionTotalsReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewTransactionTotalsHandler(reader).Register(api)
	return api
}

func TestHTTP_TransactionTotals_HappyPath(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockReader := new(mockTransactionTotalsReader)
	mockReader.On("TotalsByMonthAndCategory", mock.Anything, 1, 2025, 2, 2025).
		Return([]transaction.MonthTotals{
			{
				Year:  2025,
				Month: 1,
				Categories: []transaction.CategoryTotal{
					{
						CategoryID:   catID,
						CategoryName: "Food",
						Total:        decimal.RequireFromString("42.50"),
					},
				},
			},
			{
				Year:       2025,
				Month:      2,
				Categories: []transaction.CategoryTotal{},
			},
		}, nil)

	path := "/v1/transaction/totals?startMonth=1&startYear=2025&endMonth=2&endYear=2025"
	resp := newTotalsTestAPI(t, mockReader).Get(path)

	assert.Equal(t, http.StatusOK, resp.Code)
	var body TransactionTotalsResponseBody
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.ByMonth, 2)
	assert.Equal(t, 2025, body.ByMonth[0].Year)
	assert.Equal(t, 1, body.ByMonth[0].Month)
	assert.Len(t, body.ByMonth[0].ByCategory, 1)
	assert.Equal(t, catID.String(), body.ByMonth[0].ByCategory[0].CategoryID)
	assert.Equal(t, "42.5", body.ByMonth[0].ByCategory[0].Total)
	assert.Equal(t, 2025, body.ByMonth[1].Year)
	assert.Equal(t, 2, body.ByMonth[1].Month)
	assert.Empty(t, body.ByMonth[1].ByCategory)
	mockReader.AssertExpectations(t)
}

func TestHTTP_TransactionTotals_InvalidRange(t *testing.T) {
	mockReader := new(mockTransactionTotalsReader)

	path := "/v1/transaction/totals?startMonth=6&startYear=2025&endMonth=3&endYear=2025"
	resp := newTotalsTestAPI(t, mockReader).Get(path)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockReader.AssertNotCalled(t, "TotalsByMonthAndCategory")
}

func TestHTTP_TransactionTotals_ServiceError(t *testing.T) {
	mockReader := new(mockTransactionTotalsReader)
	mockReader.On("TotalsByMonthAndCategory", mock.Anything, 1, 2025, 1, 2025).
		Return(([]transaction.MonthTotals)(nil), errors.New("database unavailable"))

	path := "/v1/transaction/totals?startMonth=1&startYear=2025&endMonth=1&endYear=2025"
	resp := newTotalsTestAPI(t, mockReader).Get(path)

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockReader.AssertExpectations(t)
}
