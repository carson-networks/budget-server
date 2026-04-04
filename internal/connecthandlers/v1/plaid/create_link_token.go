package v1Plaid

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid"
)

// CreateLinkToken implements budget.v1.PlaidService.CreateLinkToken.
func (s *Service) CreateLinkToken(ctx context.Context, req *connect.Request[plaid.CreateLinkTokenRequest]) (*connect.Response[plaid.CreateLinkTokenResponse], error) {
	_ = req
	token, expiration, err := s.PlaidClient.CreateLinkToken(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&plaid.CreateLinkTokenResponse{
		LinkToken:  token,
		Expiration: timestamppb.New(expiration),
	}), nil
}
