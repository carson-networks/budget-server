package account

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storageaccount "github.com/carson-networks/budget-server/internal/storage/account"
)

// ListAccounts implements budget.v1.AccountService.ListAccounts.
func (s *Service) ListAccounts(ctx context.Context, req *connect.Request[budgetv1.ListAccountsRequest]) (*connect.Response[budgetv1.ListAccountsResponse], error) {
	logData := logging.GetLogData(ctx)
	limit := 20
	offset := 0
	if c := req.Msg.GetCursor(); c != nil {
		if c.GetPosition() < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cursor position must be non-negative"))
		}
		offset = int(c.GetPosition())
		if c.GetLimit() > 0 {
			limit = int(c.GetLimit())
		}
	}
	filter := &storageaccount.AccountFilter{
		Limit:  limit,
		Offset: offset,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listAccountsMs")
	}
	result, err := s.Storage.Read().Accounts.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	accounts := result.Accounts
	if accounts == nil {
		accounts = []*storageaccount.Account{}
	}
	if logData != nil {
		logData.AddData("accountCount", len(accounts))
	}

	out := &budgetv1.ListAccountsResponse{
		Accounts: make([]*budgetv1.Account, len(accounts)),
	}
	for i, acc := range accounts {
		out.Accounts[i] = &budgetv1.Account{
			Id:              acc.ID.String(),
			Name:            acc.Name,
			Type:            budgetv1.AccountType(acc.Type),
			SubType:         acc.SubType,
			Balance:         acc.Balance.String(),
			StartingBalance: acc.StartingBalance.String(),
			CreatedAt:       timestamppb.New(acc.CreatedAt),
		}
	}
	if result.NextCursor != nil {
		out.NextCursor = &budgetv1.ListAccountsCursor{
			Position: int32(result.NextCursor.Position),
			Limit:    int32(result.NextCursor.Limit),
		}
	}
	return connect.NewResponse(out), nil
}
