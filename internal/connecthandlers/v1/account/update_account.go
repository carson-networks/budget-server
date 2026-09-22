package v1Account

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// UpdateAccount implements account.v1.AccountService.UpdateAccount.
func (s *Service) UpdateAccount(ctx context.Context, req *connect.Request[account.UpdateAccountRequest]) (*connect.Response[account.UpdateAccountResponse], error) {
	id, err := uuid.FromString(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var name *string
	if req.Msg.Name != nil {
		name = req.Msg.Name
	}
	var subType *string
	if req.Msg.SubType != nil {
		subType = req.Msg.SubType
	}
	var startingBalance *decimal.Decimal
	if req.Msg.StartingBalance != nil {
		parsed, perr := decimal.NewFromString(*req.Msg.StartingBalance)
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr)
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
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&account.UpdateAccountResponse{}), nil
}
