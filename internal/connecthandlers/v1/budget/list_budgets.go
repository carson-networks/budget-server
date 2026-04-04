package v1Budget

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget"
	"github.com/carson-networks/budget-server/internal/logging"
	storagebudget "github.com/carson-networks/budget-server/internal/storage/budget"
)

// ListBudgets implements budget.v1.BudgetService.ListBudgets.
func (s *Service) ListBudgets(ctx context.Context, req *connect.Request[budget.ListBudgetsRequest]) (*connect.Response[budget.ListBudgetsResponse], error) {
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
	return connect.NewResponse(out), nil
}
