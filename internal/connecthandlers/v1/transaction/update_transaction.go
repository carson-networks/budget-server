package v1Transaction

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

func (s *Service) UpdateTransaction(ctx context.Context, req *transaction.UpdateTransactionRequest) (*transaction.UpdateTransactionResponse, error) {
	id, err := uuid.FromString(req.GetId())
	if err != nil || id == uuid.Nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, "invalid transaction id")
	}
	if req.CategoryId == nil && req.Amount == nil && req.TransactionName == nil && req.TransactionDate == nil && req.MerchantName == nil && req.NeedsReview == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, "at least one field must be supplied")
	}
	action := &actions.UpdateTransaction{
		ID: id, TransactionName: req.TransactionName, MerchantName: req.MerchantName, NeedsReview: req.NeedsReview,
	}
	if req.CategoryId != nil {
		categoryID, err := uuid.FromString(*req.CategoryId)
		if err != nil || categoryID == uuid.Nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, "invalid category id")
		}
		action.CategoryID = &categoryID
	}
	if req.Amount != nil {
		amount, err := decimal.NewFromString(*req.Amount)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		}
		if !amount.Equal(amount.Round(4)) {
			return nil, connect.NewError(connect.CodeInvalidArgument, "amount must have at most four decimal places")
		}
		action.Amount = &amount
	}
	if req.TransactionDate != nil {
		if err := req.TransactionDate.CheckValid(); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		}
		date := req.TransactionDate.AsTime()
		action.TransactionDate = &date
	}
	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrTransactionNotFound), errors.Is(err, actions.ErrCategoryNotFoundForTransaction), errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrCategoryDisabled), errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &transaction.UpdateTransactionResponse{}, nil
}
