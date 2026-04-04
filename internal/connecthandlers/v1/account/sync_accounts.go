package v1Account

import (
	"context"

	"connectrpc.com/connect"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/account"
	"github.com/gofrs/uuid/v5"
)

func (s *Service) SyncAccounts(ctx context.Context, req *connect.Request[account.SyncAccountsRequest]) (*connect.Response[account.SyncAccountsResponse], error) {
	accountUUIDs := make([]uuid.UUID, 0, len(req.Msg.GetAccountIds()))
	for _, idStr := range req.Msg.GetAccountIds() {
		id, err := uuid.FromString(idStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		accountUUIDs = append(accountUUIDs, id)
	}
	if err := s.Orchestrator.Sync(ctx, accountUUIDs); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&account.SyncAccountsResponse{}), nil
}
