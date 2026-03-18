package budget

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/storage/budget"
)

type mockBudgetReader struct {
	mock.Mock
}

func (m *mockBudgetReader) ListForRange(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]*budget.Budget, error) {
	args := m.Called(ctx, startMonth, startYear, endMonth, endYear)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*budget.Budget), args.Error(1)
}

func newListBudgetsTestAPI(t *testing.T, reader budgetReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewListBudgetsHandler(reader).Register(api)
	return api
}

func TestHTTP_ListBudgets_Empty(t *testing.T) {
	mockReader := &mockBudgetReader{}
	mockReader.On("ListForRange", mock.Anything, 1, 2025, 3, 2025).
		Return([]*budget.Budget{}, nil)

	resp := newListBudgetsTestAPI(t, mockReader).Get("/v1/budgets?startMonth=1&startYear=2025&endMonth=3&endYear=2025")

	assert.Equal(t, http.StatusOK, resp.Code)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListBudgets_NonEmpty(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockReader := &mockBudgetReader{}
	mockReader.On("ListForRange", mock.Anything, 1, 2025, 3, 2025).
		Return([]*budget.Budget{
			{
				CategoryID: catID,
				Month:      1,
				Year:       2025,
				Amount:     decimal.RequireFromString("100.50"),
			},
		}, nil)

	resp := newListBudgetsTestAPI(t, mockReader).Get("/v1/budgets?startMonth=1&startYear=2025&endMonth=3&endYear=2025")

	assert.Equal(t, http.StatusOK, resp.Code)
	var body struct {
		Budgets []struct {
			CategoryID string `json:"categoryID"`
			Month      int    `json:"month"`
			Year       int    `json:"year"`
			Amount     string `json:"amount"`
		} `json:"budgets"`
	}
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Len(t, body.Budgets, 1)
	assert.Equal(t, catID.String(), body.Budgets[0].CategoryID)
	assert.Equal(t, 1, body.Budgets[0].Month)
	assert.Equal(t, 2025, body.Budgets[0].Year)
	assert.Equal(t, "100.5", body.Budgets[0].Amount)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListBudgets_InvalidStartMonth(t *testing.T) {
	mockReader := &mockBudgetReader{}

	resp := newListBudgetsTestAPI(t, mockReader).Get("/v1/budgets?startMonth=0&startYear=2025&endMonth=3&endYear=2025")

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockReader.AssertNotCalled(t, "ListForRange")
}

func TestHTTP_ListBudgets_InvalidEndMonth(t *testing.T) {
	mockReader := &mockBudgetReader{}

	resp := newListBudgetsTestAPI(t, mockReader).Get("/v1/budgets?startMonth=1&startYear=2025&endMonth=13&endYear=2025")

	assert.Equal(t, http.StatusUnprocessableEntity, resp.Code)
	mockReader.AssertNotCalled(t, "ListForRange")
}

func TestHTTP_ListBudgets_EndBeforeStart(t *testing.T) {
	mockReader := &mockBudgetReader{}

	resp := newListBudgetsTestAPI(t, mockReader).Get("/v1/budgets?startMonth=3&startYear=2025&endMonth=1&endYear=2025")

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockReader.AssertNotCalled(t, "ListForRange")
}
