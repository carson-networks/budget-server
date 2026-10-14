package v1Transaction

import (
	"context"
	"errors"
	"net/http"
	"time"

	"connectrpc.com/connect/v2"

	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

// CreateTransaction implements transaction.v1.TransactionService.CreateTransaction.
func (s *Service) CreateTransaction(ctx context.Context, req *transaction.CreateTransactionRequest) (*transaction.CreateTransactionResponse, error) {
	accountID, err := uuid.FromString(req.GetAccountId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	categoryID, err := uuid.FromString(req.GetCategoryId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}
	amount, err := decimal.NewFromString(req.GetAmount())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	var transactionDate time.Time
	if ts := req.GetTransactionDate(); ts != nil {
		transactionDate = ts.AsTime()
	} else {
		transactionDate = time.Now()
	}

	action := &actions.CreateTransaction{
		AccountID:       accountID,
		CategoryID:      &categoryID,
		Amount:          amount,
		TransactionName: req.GetTransactionName(),
		TransactionDate: transactionDate,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrCategoryNotFoundForTransaction):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrCategoryDisabled):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &transaction.CreateTransactionResponse{Status: http.StatusCreated}, nil
}
