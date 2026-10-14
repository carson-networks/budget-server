package v1Account

import (
	"context"

	"connectrpc.com/connect/v2"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	"github.com/gofrs/uuid/v5"
)

func (s *Service) SyncAccounts(ctx context.Context, req *account.SyncAccountsRequest) (*account.SyncAccountsResponse, error) {
	accountUUIDs := make([]uuid.UUID, 0, len(req.GetAccountIds()))
	for _, idStr := range req.GetAccountIds() {
		id, err := uuid.FromString(idStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		}
		accountUUIDs = append(accountUUIDs, id)
	}
	if err := s.Orchestrator.Sync(ctx, accountUUIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
	}
	return &account.SyncAccountsResponse{}, nil
}
