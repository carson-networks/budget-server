package v1Budget

import (
	"context"

	"connectrpc.com/connect/v2"

	budget "github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagebudget "github.com/carson-networks/budget-server/internal/storage/budget"
)

// ListBudgets implements budget.v1.BudgetService.ListBudgets.
func (s *Service) ListBudgets(ctx context.Context, req *budget.ListBudgetsRequest) (*budget.ListBudgetsResponse, error) {
	logData := logging.GetLogData(ctx)
	sm, sy := int(req.GetStartMonth()), int(req.GetStartYear())
	em, ey := int(req.GetEndMonth()), int(req.GetEndYear())
	if ey < sy || (ey == sy && em < sm) {
		return nil, connect.NewError(connect.CodeInvalidArgument, "end date must be >= start date")
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listBudgetsMs")
	}
	budgets, err := s.Storage.Read().Budgets.ListForRange(ctx, sm, sy, em, ey)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}
	if budgets == nil {
		budgets = []*storagebudget.Budget{}
	}
	if logData != nil {
		logData.AddData("budgetCount", len(budgets))
	}

	out := &budget.ListBudgetsResponse{
		Budgets: make([]*budget.Budget, len(budgets)),
	}
	for i, b := range budgets {
		out.Budgets[i] = &budget.Budget{
			CategoryId: b.CategoryID.String(),
			Month:      int32(b.Month),
			Year:       int32(b.Year),
			Amount:     b.Amount.String(),
		}
	}
	return out, nil
}
