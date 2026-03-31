package account

import (
	"context"
	"net/http"

	"connectrpc.com/connect"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	storageaccount "github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/shopspring/decimal"
)

// CreateAccount implements budget.v1.AccountService.CreateAccount.
func (s *Service) CreateAccount(ctx context.Context, req *connect.Request[budgetv1.CreateAccountRequest]) (*connect.Response[budgetv1.CreateAccountResponse], error) {
	startingBalance, err := decimal.NewFromString(req.Msg.GetStartingBalance())
	if err != nil {
		startingBalance = decimal.Zero
	}
	action := &actions.CreateAccount{
		Name:            req.Msg.GetName(),
		Type:            storageaccount.AccountType(req.Msg.GetType()),
		SubType:         req.Msg.GetSubType(),
		StartingBalance: startingBalance,
	}
	if err := s.Operator.Process(ctx, action); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&budgetv1.CreateAccountResponse{Status: http.StatusCreated}), nil
}
