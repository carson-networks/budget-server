package v1Plaid

import (
	"context"
	"net/http"

	"connectrpc.com/connect/v2"

	plaid "github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid/v1"
	v1Account "github.com/carson-networks/budget-server/internal/connecthandlers/v1/account"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/shopspring/decimal"
)

// ExchangeToken implements plaid.v1.PlaidService.ExchangeToken.
func (s *Service) ExchangeToken(ctx context.Context, req *plaid.ExchangeTokenRequest) (*plaid.ExchangeTokenResponse, error) {
	if len(req.GetAccounts()) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, "at least one account must be selected")
	}

	accessToken, plaidItemID, err := s.PlaidClient.ExchangePublicToken(ctx, req.GetPublicToken())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}

	plaidAccounts := make([]actions.PlaidAccountToLink, len(req.GetAccounts()))
	for i, a := range req.GetAccounts() {
		bal, derr := decimal.NewFromString(a.GetBalance())
		if derr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, derr.Error()).WithCause(derr)
		}
		accType, terr := v1Account.FromConnectAccountType(a.GetType())
		if terr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, terr.Error()).WithCause(terr)
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
		InstitutionID:   req.GetInstitutionId(),
		InstitutionName: req.GetInstitutionName(),
		Accounts:        plaidAccounts,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}

	if err := s.Orchestrator.Sync(ctx, action.CreatedAccountIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}

	return &plaid.ExchangeTokenResponse{Status: http.StatusCreated}, nil
}
