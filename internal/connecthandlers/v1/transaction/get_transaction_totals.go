package v1Transaction

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

// GetTransactionTotals implements transaction.v1.TransactionService.GetTransactionTotals.
func (s *Service) GetTransactionTotals(ctx context.Context, req *connect.Request[transaction.GetTransactionTotalsRequest]) (*connect.Response[transaction.GetTransactionTotalsResponse], error) {
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

	out := &transaction.GetTransactionTotalsResponse{
		ByMonth: make([]*transaction.TransactionTotalsMonth, len(months)),
	}
	for i, m := range months {
		cats := make([]*transaction.TransactionTotalsCategory, 0, len(m.Categories))
		for _, c := range m.Categories {
			cats = append(cats, &transaction.TransactionTotalsCategory{
				CategoryId: c.CategoryID.String(),
				Total:      c.Total.String(),
			})
		}
		out.ByMonth[i] = &transaction.TransactionTotalsMonth{
			Year:       int32(m.Year),
			Month:      int32(m.Month),
			ByCategory: cats,
		}
	}
	return connect.NewResponse(out), nil
}
