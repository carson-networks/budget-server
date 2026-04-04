package v1Plaid

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	plaid "github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid/v1"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/account"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/shopspring/decimal"
)

// ExchangeToken implements plaid.v1.PlaidService.ExchangeToken.
func (s *Service) ExchangeToken(ctx context.Context, req *connect.Request[plaid.ExchangeTokenRequest]) (*connect.Response[plaid.ExchangeTokenResponse], error) {
	if len(req.Msg.GetAccounts()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one account must be selected"))
	}

	accessToken, plaidItemID, err := s.PlaidClient.ExchangePublicToken(ctx, req.Msg.GetPublicToken())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	plaidAccounts := make([]actions.PlaidAccountToLink, len(req.Msg.GetAccounts()))
	for i, a := range req.Msg.GetAccounts() {
		bal, derr := decimal.NewFromString(a.GetBalance())
		if derr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, derr)
		}
		accType, terr := v1Account.FromConnectAccountType(a.GetType())
		if terr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, terr)
		}
		plaidAccounts[i] = actions.PlaidAccountToLink{
			PlaidAccountID: a.GetPlaidAccountId(),
			Name:           a.GetName(),
			Type:           accType,
			SubType:        a.GetSubType(),
			Balance:        bal,
		}
	}

	action := &actions.LinkPlaidItem{
		AccessToken:     accessToken,
		PlaidItemID:     plaidItemID,
		InstitutionID:   req.Msg.GetInstitutionId(),
		InstitutionName: req.Msg.GetInstitutionName(),
		Accounts:        plaidAccounts,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.Orchestrator.Sync(ctx, action.CreatedAccountIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&plaid.ExchangeTokenResponse{Status: http.StatusCreated}), nil
}
