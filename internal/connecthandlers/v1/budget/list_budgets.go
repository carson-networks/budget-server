package budget

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagebudget "github.com/carson-networks/budget-server/internal/storage/budget"
)

// ListBudgets implements budget.v1.BudgetService.ListBudgets.
func (s *Service) ListBudgets(ctx context.Context, req *connect.Request[budgetv1.ListBudgetsRequest]) (*connect.Response[budgetv1.ListBudgetsResponse], error) {
	logData := logging.GetLogData(ctx)
	sm, sy := int(req.Msg.GetStartMonth()), int(req.Msg.GetStartYear())
	em, ey := int(req.Msg.GetEndMonth()), int(req.Msg.GetEndYear())
	if ey < sy || (ey == sy && em < sm) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("end date must be >= start date"))
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
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if budgets == nil {
		budgets = []*storagebudget.Budget{}
	}
	if logData != nil {
		logData.AddData("budgetCount", len(budgets))
	}

	out := &budgetv1.ListBudgetsResponse{
		Budgets: make([]*budgetv1.BudgetRow, len(budgets)),
	}
	for i, b := range budgets {
		out.Budgets[i] = &budgetv1.BudgetRow{
			CategoryId: b.CategoryID.String(),
			Month:      int32(b.Month),
			Year:       int32(b.Year),
			Amount:     b.Amount.String(),
		}
	}
	return connect.NewResponse(out), nil
}
