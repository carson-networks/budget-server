package v1Plaid

import (
	"context"

	"connectrpc.com/connect/v2"
	"google.golang.org/protobuf/types/known/timestamppb"

	plaid "github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid/v1"
)

// CreateLinkToken implements plaid.v1.PlaidService.CreateLinkToken.
func (s *Service) CreateLinkToken(ctx context.Context, req *plaid.CreateLinkTokenRequest) (*plaid.CreateLinkTokenResponse, error) {
	_ = req
	token, expiration, err := s.PlaidClient.CreateLinkToken(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}
	return &plaid.CreateLinkTokenResponse{
		LinkToken:  token,
		Expiration: timestamppb.New(expiration),
	}, nil
}
