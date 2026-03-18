package budget

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
)

// SetBudgetBody is the request body for setting a budget.
type SetBudgetBody struct {
	CategoryID            string `json:"categoryID" required:"true" doc:"Category UUID (must be a leaf category)"`
	Month                 int    `json:"month" required:"true" minimum:"1" maximum:"12" doc:"Month (1-12)"`
	Year                  int    `json:"year" required:"true" doc:"Year"`
	Amount                string `json:"amount" required:"true" doc:"Budget amount"`
	OverwriteFutureMonths bool   `json:"overwriteFutureMonths" doc:"If true, delete budgets for this category in months after the specified month"`
}

// SetBudgetInput is the Huma input for setting a budget.
type SetBudgetInput struct {
	Body SetBudgetBody
}

// SetBudgetResponseBody is the response body for setting a budget.
type SetBudgetResponseBody struct {
	CategoryID string `json:"categoryID" doc:"Category UUID"`
	Month      int    `json:"month" doc:"Month (1-12)"`
	Year       int    `json:"year" doc:"Year"`
	Amount     string `json:"amount" doc:"Budget amount"`
}

// SetBudgetOutput is the Huma output for setting a budget.
type SetBudgetOutput struct {
	Body SetBudgetResponseBody
}

// SetBudgetHandler handles POST /v1/budgets.
type SetBudgetHandler struct {
	Operator operator.IProcessor
}

// NewSetBudgetHandler creates a new SetBudgetHandler.
func NewSetBudgetHandler(op operator.IProcessor) *SetBudgetHandler {
	return &SetBudgetHandler{Operator: op}
}

// Register registers the set budget endpoint with the Huma API.
func (h *SetBudgetHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "set-budget",
		Method:      http.MethodPost,
		Path:        "/v1/budgets",
		Summary:     "Set budget",
		Description: "Create or overwrite a budget for a category for the specified month. When overwriteFutureMonths is true, deletes budgets for that category in months after the specified month.",
		Tags:        []string{"Budgets"},
	}, h.handle)
}

func (h *SetBudgetHandler) handle(ctx context.Context, input *SetBudgetInput) (*SetBudgetOutput, error) {
	categoryID, err := uuid.FromString(input.Body.CategoryID)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid categoryID", err)
	}
	amount, err := decimal.NewFromString(input.Body.Amount)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid amount", err)
	}

	action := &actions.SetBudget{
		CategoryID:            categoryID,
		Month:                 input.Body.Month,
		Year:                  input.Body.Year,
		Amount:                amount,
		OverwriteFutureMonths: input.Body.OverwriteFutureMonths,
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrCategoryNotFoundForBudget):
			return nil, huma.NewError(http.StatusNotFound, "category not found", err)
		case errors.Is(err, actions.ErrCategoryIsParent):
			return nil, huma.NewError(http.StatusBadRequest, "category is a group; use a leaf category", err)
		case errors.Is(err, actions.ErrInvalidMonth):
			return nil, huma.NewError(http.StatusBadRequest, "month must be between 1 and 12", err)
		default:
			return nil, huma.NewError(http.StatusInternalServerError, "failed to set budget", err)
		}
	}

	return &SetBudgetOutput{
		Body: SetBudgetResponseBody{
			CategoryID: categoryID.String(),
			Month:      input.Body.Month,
			Year:       input.Body.Year,
			Amount:     amount.String(),
		},
	}, nil
}
