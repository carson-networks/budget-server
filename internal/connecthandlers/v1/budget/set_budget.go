package v1Budget

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"

	budget "github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// SetBudget implements budget.v1.BudgetService.SetBudget.
func (s *Service) SetBudget(ctx context.Context, req *budget.SetBudgetRequest) (*budget.SetBudgetResponse, error) {
	categoryID, err := uuid.FromString(req.GetCategoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	action := &actions.SetBudget{
		CategoryID:            categoryID,
		Month:                 int(req.GetMonth()),
		Year:                  int(req.GetYear()),
		Amount:                amount,
		OverwriteFutureMonths: req.GetOverwriteFutureMonths(),
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrCategoryNotFoundForBudget):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrInvalidMonth):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}

	return &budget.SetBudgetResponse{}, nil
}
