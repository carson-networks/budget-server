package v1Budget

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// SetBudget implements budget.v1.BudgetService.SetBudget.
func (s *Service) SetBudget(ctx context.Context, req *connect.Request[budget.SetBudgetRequest]) (*connect.Response[budget.SetBudgetResponse], error) {
	categoryID, err := uuid.FromString(req.Msg.GetCategoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	amount, err := decimal.NewFromString(req.Msg.GetAmount())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	action := &actions.SetBudget{
		CategoryID:            categoryID,
		Month:                 int(req.Msg.GetMonth()),
		Year:                  int(req.Msg.GetYear()),
		Amount:                amount,
		OverwriteFutureMonths: req.Msg.GetOverwriteFutureMonths(),
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrCategoryNotFoundForBudget):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, actions.ErrInvalidMonth):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}

	return connect.NewResponse(&budget.SetBudgetResponse{}), nil
}
