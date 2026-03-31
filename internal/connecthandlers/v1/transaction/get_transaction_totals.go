package transaction

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

// GetTransactionTotals implements budget.v1.TransactionService.GetTransactionTotals.
func (s *Service) GetTransactionTotals(ctx context.Context, req *connect.Request[budgetv1.GetTransactionTotalsRequest]) (*connect.Response[budgetv1.GetTransactionTotalsResponse], error) {
	logData := logging.GetLogData(ctx)
	sm, sy := int(req.Msg.GetStartMonth()), int(req.Msg.GetStartYear())
	em, ey := int(req.Msg.GetEndMonth()), int(req.Msg.GetEndYear())
	if ey < sy || (ey == sy && em < sm) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("end date must be >= start date"))
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("transactionTotalsMs")
	}
	months, err := s.Storage.Read().Transactions.TotalsByMonthAndCategory(ctx, sm, sy, em, ey)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if months == nil {
		months = []storagetransaction.MonthTotals{}
	}
	if logData != nil {
		logData.AddData("monthCount", len(months))
	}

	out := &budgetv1.GetTransactionTotalsResponse{
		ByMonth: make([]*budgetv1.TransactionTotalsMonth, len(months)),
	}
	for i, m := range months {
		cats := make([]*budgetv1.TransactionTotalsCategory, 0, len(m.Categories))
		for _, c := range m.Categories {
			cats = append(cats, &budgetv1.TransactionTotalsCategory{
				CategoryId: c.CategoryID.String(),
				Total:      c.Total.String(),
			})
		}
		out.ByMonth[i] = &budgetv1.TransactionTotalsMonth{
			Year:       int32(m.Year),
			Month:      int32(m.Month),
			ByCategory: cats,
		}
	}
	return connect.NewResponse(out), nil
}
