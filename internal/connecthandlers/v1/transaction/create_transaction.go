package v1Transaction

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// CreateTransaction implements budget.v1.TransactionService.CreateTransaction.
func (s *Service) CreateTransaction(ctx context.Context, req *connect.Request[transaction.CreateTransactionRequest]) (*connect.Response[transaction.CreateTransactionResponse], error) {
	accountID, err := uuid.FromString(req.Msg.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	categoryID, err := uuid.FromString(req.Msg.GetCategoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	amount, err := decimal.NewFromString(req.Msg.GetAmount())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var transactionDate time.Time
	if ts := req.Msg.GetTransactionDate(); ts != nil {
		transactionDate = ts.AsTime()
	} else {
		transactionDate = time.Now()
	}

	action := &actions.CreateTransaction{
		AccountID:       accountID,
		CategoryID:      &categoryID,
		Amount:          amount,
		TransactionName: req.Msg.GetTransactionName(),
		TransactionDate: transactionDate,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrCategoryNotFoundForTransaction):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrCategoryDisabled):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&transaction.CreateTransactionResponse{Status: http.StatusCreated}), nil
}
