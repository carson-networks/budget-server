package v1Transaction

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/logging"
	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

// ListTransactions implements transaction.v1.TransactionService.ListTransactions.
func (s *Service) ListTransactions(ctx context.Context, req *connect.Request[transaction.ListTransactionsRequest]) (*connect.Response[transaction.ListTransactionsResponse], error) {
	logData := logging.GetLogData(ctx)
	limit := 20
	offset := 0
	var maxCreationTime *time.Time

	if c := req.Msg.GetCursor(); c != nil {
		if c.GetPosition() < 0 {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("cursor position must be non-negative"))
		}
		offset = int(c.GetPosition())
		if c.GetLimit() > 0 {
			limit = int(c.GetLimit())
		}
		if ts := c.GetMaxCreationTime(); ts != nil {
			t := ts.AsTime()
			maxCreationTime = &t
		}
	}

	filter := &storagetransaction.TransactionFilter{
		Limit:           limit,
		Offset:          offset,
		MaxCreationTime: maxCreationTime,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listTransactionsMs")
	}
	result, err := s.Storage.Read().Transactions.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	transactions := result.Transactions
	if transactions == nil {
		transactions = []*storagetransaction.Transaction{}
	}
	if logData != nil {
		logData.AddData("transactionCount", len(transactions))
	}

	out := &transaction.ListTransactionsResponse{
		Transactions: make([]*transaction.Transaction, len(transactions)),
	}
	for i, tx := range transactions {
		t := &transaction.Transaction{
			Id:              tx.ID.String(),
			AccountId:       tx.AccountID.String(),
			Amount:          tx.Amount.String(),
			TransactionName: tx.TransactionName,
			TransactionDate: timestamppb.New(tx.TransactionDate),
			CreatedAt:       timestamppb.New(tx.CreatedAt),
		}
		if tx.CategoryID != nil {
			cid := tx.CategoryID.String()
			t.CategoryId = &cid
		}
		out.Transactions[i] = t
	}
	if result.NextCursor != nil {
		out.NextCursor = &transaction.ListTransactionsCursor{
			Position:        int32(result.NextCursor.Position),
			Limit:           int32(result.NextCursor.Limit),
			MaxCreationTime: timestamppb.New(result.NextCursor.MaxCreationTime),
		}
	}
	return connect.NewResponse(out), nil
}
