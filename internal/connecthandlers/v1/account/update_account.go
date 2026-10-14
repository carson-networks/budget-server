package v1Account

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// UpdateAccount implements account.v1.AccountService.UpdateAccount.
func (s *Service) UpdateAccount(ctx context.Context, req *account.UpdateAccountRequest) (*account.UpdateAccountResponse, error) {
	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	var name *string
	if req.Name != nil {
		name = req.Name
	}
	var subType *string
	if req.SubType != nil {
		subType = req.SubType
	}
	var startingBalance *decimal.Decimal
	if req.StartingBalance != nil {
		parsed, perr := decimal.NewFromString(*req.StartingBalance)
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr.Error()).WithCause(perr)
		}
		startingBalance = &parsed
	}

	action := &actions.UpdateAccount{
		ID:              id,
		Name:            name,
		SubType:         subType,
		StartingBalance: startingBalance,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &account.UpdateAccountResponse{}, nil
}
