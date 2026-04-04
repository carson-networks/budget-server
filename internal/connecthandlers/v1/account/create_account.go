package v1Account

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/shopspring/decimal"
)

// CreateAccount implements account.v1.AccountService.CreateAccount.
func (s *Service) CreateAccount(ctx context.Context, req *connect.Request[account.CreateAccountRequest]) (*connect.Response[account.CreateAccountResponse], error) {
	accType, err := FromConnectAccountType(req.Msg.GetType())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	startingBalance, err := decimal.NewFromString(req.Msg.GetStartingBalance())
	if err != nil {
		startingBalance = decimal.Zero
	}
	action := &actions.CreateAccount{
		Name:            req.Msg.GetName(),
		Type:            accType,
		SubType:         req.Msg.GetSubType(),
		StartingBalance: startingBalance,
	}
	if err := s.Operator.Process(ctx, action); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&account.CreateAccountResponse{Status: http.StatusCreated}), nil
}
