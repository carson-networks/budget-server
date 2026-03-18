package budget

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/storage/budget"
)

// Budget is the API response model for a budget.
type Budget struct {
	CategoryID string `json:"categoryID" doc:"Category UUID"`
	Month      int    `json:"month" doc:"Month (1-12)"`
	Year       int    `json:"year" doc:"Year"`
	Amount     string `json:"amount" doc:"Budget amount"`
}

// ListBudgetsInput is the Huma input for listing budgets.
type ListBudgetsInput struct {
	StartMonth int `query:"startMonth" required:"true" minimum:"1" maximum:"12" doc:"Start month (1-12)"`
	StartYear  int `query:"startYear" required:"true" doc:"Start year"`
	EndMonth   int `query:"endMonth" required:"true" minimum:"1" maximum:"12" doc:"End month (1-12)"`
	EndYear    int `query:"endYear" required:"true" doc:"End year"`
}

// ListBudgetsResponseBody is the response body for listing budgets.
type ListBudgetsResponseBody struct {
	Budgets []Budget `json:"budgets" doc:"Budget rows for the time range"`
}

// ListBudgetsOutput is the Huma output for listing budgets.
type ListBudgetsOutput struct {
	Body ListBudgetsResponseBody
}

type budgetReader interface {
	ListForRange(ctx context.Context, startMonth, startYear, endMonth, endYear int) ([]*budget.Budget, error)
}

// ListBudgetsHandler handles GET /v1/budgets.
type ListBudgetsHandler struct {
	BudgetReader budgetReader
}

// NewListBudgetsHandler creates a new ListBudgetsHandler.
func NewListBudgetsHandler(reader budgetReader) *ListBudgetsHandler {
	return &ListBudgetsHandler{BudgetReader: reader}
}

// Register registers the list budgets endpoint with the Huma API.
func (h *ListBudgetsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-budgets",
		Method:      http.MethodGet,
		Path:        "/v1/budgets",
		Summary:     "List budgets",
		Description: "Returns budget rows for a time range. For each category, at most one row at or before startMonth (the latest such), plus all rows where month is > startMonth and <= endMonth.",
		Tags:        []string{"Budgets"},
	}, h.handle)
}

func (h *ListBudgetsHandler) handle(ctx context.Context, input *ListBudgetsInput) (*ListBudgetsOutput, error) {
	logData := logging.GetLogData(ctx)

	if input.EndYear < input.StartYear || (input.EndYear == input.StartYear && input.EndMonth < input.StartMonth) {
		return nil, huma.NewError(http.StatusBadRequest, "end date must be >= start date", nil)
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listBudgetsMs")
	}
	budgets, err := h.BudgetReader.ListForRange(ctx, input.StartMonth, input.StartYear, input.EndMonth, input.EndYear)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list budgets", err)
	}

	if budgets == nil {
		budgets = []*budget.Budget{}
	}

	if logData != nil {
		logData.AddData("budgetCount", len(budgets))
	}

	resp := ListBudgetsResponseBody{
		Budgets: make([]Budget, len(budgets)),
	}
	for i, b := range budgets {
		resp.Budgets[i] = Budget{
			CategoryID: b.CategoryID.String(),
			Month:      b.Month,
			Year:       b.Year,
			Amount:     b.Amount.String(),
		}
	}

	return &ListBudgetsOutput{Body: resp}, nil
}
