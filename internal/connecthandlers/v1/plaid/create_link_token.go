package plaid

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
)

// CreateLinkToken implements budget.v1.PlaidService.CreateLinkToken.
func (s *Service) CreateLinkToken(ctx context.Context, req *connect.Request[budgetv1.CreateLinkTokenRequest]) (*connect.Response[budgetv1.CreateLinkTokenResponse], error) {
	_ = req
	token, expiration, err := s.PlaidClient.CreateLinkToken(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&budgetv1.CreateLinkTokenResponse{
		LinkToken:  token,
		Expiration: timestamppb.New(expiration),
	}), nil
}
