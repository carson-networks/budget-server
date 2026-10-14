package v1Account

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
)

// DeleteAccount implements account.v1.AccountService.DeleteAccount.
func (s *Service) DeleteAccount(ctx context.Context, req *account.DeleteAccountRequest) (*account.DeleteAccountResponse, error) {
	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	action := &actions.DeleteAccount{ID: id}
	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &account.DeleteAccountResponse{}, nil
}
