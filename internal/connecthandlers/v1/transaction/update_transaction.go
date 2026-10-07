package v1Transaction

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
)

func (s *Service) UpdateTransaction(ctx context.Context, req *connect.Request[transaction.UpdateTransactionRequest]) (*connect.Response[transaction.UpdateTransactionResponse], error) {
	id, err := uuid.FromString(req.Msg.GetId())
	if err != nil || id == uuid.Nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid transaction id"))
	}
	if req.Msg.CategoryId == nil && req.Msg.Amount == nil && req.Msg.TransactionName == nil && req.Msg.TransactionDate == nil && req.Msg.MerchantName == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("at least one field must be supplied"))
	}
	action := &actions.UpdateTransaction{
		ID: id, TransactionName: req.Msg.TransactionName, MerchantName: req.Msg.MerchantName,
	}
	if req.Msg.CategoryId != nil {
		categoryID, err := uuid.FromString(*req.Msg.CategoryId)
		if err != nil || categoryID == uuid.Nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid category id"))
		}
		action.CategoryID = &categoryID
	}
	if req.Msg.Amount != nil {
		amount, err := decimal.NewFromString(*req.Msg.Amount)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		if !amount.Equal(amount.Round(4)) {
			return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("amount must have at most four decimal places"))
		}
		action.Amount = &amount
	}
	if req.Msg.TransactionDate != nil {
		if err := req.Msg.TransactionDate.CheckValid(); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		date := req.Msg.TransactionDate.AsTime()
		action.TransactionDate = &date
	}
	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrTransactionNotFound), errors.Is(err, actions.ErrCategoryNotFoundForTransaction), errors.Is(err, actions.ErrAccountNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrCategoryDisabled), errors.Is(err, actions.ErrCategoryIsParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&transaction.UpdateTransactionResponse{}), nil
}
