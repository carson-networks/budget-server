package v1Account

import (
	"context"
	"net/http"

	"connectrpc.com/connect/v2"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/shopspring/decimal"
)

// CreateAccount implements account.v1.AccountService.CreateAccount.
func (s *Service) CreateAccount(ctx context.Context, req *account.CreateAccountRequest) (*account.CreateAccountResponse, error) {
	accType, err := FromConnectAccountType(req.GetType())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	startingBalance, err := decimal.NewFromString(req.GetStartingBalance())
	if err != nil {
		startingBalance = decimal.Zero
	}
	action := &actions.CreateAccount{
		Name:            req.GetName(),
		Type:            accType,
		SubType:         req.GetSubType(),
		StartingBalance: startingBalance,
	}
	if err := s.Operator.Process(ctx, action); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}
	return &account.CreateAccountResponse{Status: http.StatusCreated}, nil
}
