package budget

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
)

func newSetBudgetTestAPI(t *testing.T, op operator.IProcessor) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewSetBudgetHandler(op).Register(api)
	return api
}

func TestHTTP_SetBudget_Success(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			sb, ok := a.(*actions.SetBudget)
			return ok &&
				sb.CategoryID == catID &&
				sb.Month == 3 && sb.Year == 2025 &&
				sb.Amount.String() == "150.25" &&
				sb.OverwriteFutureMonths == false
		})).
		Return(nil)

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID:            catID.String(),
		Month:                 3,
		Year:                  2025,
		Amount:                "150.25",
		OverwriteFutureMonths: false,
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SetBudget_OverwriteFutureMonths(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			sb, ok := a.(*actions.SetBudget)
			return ok && sb.OverwriteFutureMonths == true
		})).
		Return(nil)

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID:            catID.String(),
		Month:                 1,
		Year:                  2025,
		Amount:                "200",
		OverwriteFutureMonths: true,
	})

	assert.Equal(t, http.StatusOK, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SetBudget_CategoryNotFound(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrCategoryNotFoundForBudget)

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID: catID.String(),
		Month:      3,
		Year:       2025,
		Amount:     "100",
	})

	assert.Equal(t, http.StatusNotFound, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SetBudget_CategoryIsParent(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrCategoryIsParent)

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID: catID.String(),
		Month:      3,
		Year:       2025,
		Amount:     "100",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_SetBudget_InvalidCategoryID(t *testing.T) {
	mockOp := &operator.MockIProcessor{}

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID: "not-a-uuid",
		Month:      3,
		Year:       2025,
		Amount:     "100",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}

func TestHTTP_SetBudget_InvalidAmount(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}

	resp := newSetBudgetTestAPI(t, mockOp).Post("/v1/budgets", SetBudgetBody{
		CategoryID: catID.String(),
		Month:      3,
		Year:       2025,
		Amount:     "not-a-number",
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertNotCalled(t, "Process")
}
