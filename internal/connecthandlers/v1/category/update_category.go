package category

import (
	"context"

	"connectrpc.com/connect"

	budgetv1 "github.com/carson-networks/budget-server/gen/budget/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
)

// UpdateCategory implements budget.v1.CategoryService.UpdateCategory.
func (s *Service) UpdateCategory(ctx context.Context, req *connect.Request[budgetv1.UpdateCategoryRequest]) (*connect.Response[budgetv1.UpdateCategoryResponse], error) {
	id, err := uuid.FromString(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var name *string
	if req.Msg.Name != nil {
		name = req.Msg.Name
	}
	var parentCategoryID *uuid.UUID
	if req.Msg.ParentCategoryId != nil && *req.Msg.ParentCategoryId != "" {
		pid, perr := uuid.FromString(*req.Msg.ParentCategoryId)
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr)
		}
		parentCategoryID = &pid
	}
	var isDisabled *bool
	if req.Msg.IsDisabled != nil {
		isDisabled = req.Msg.IsDisabled
	}

	action := &actions.UpdateCategory{
		ID:               id,
		Name:             name,
		ParentCategoryID: parentCategoryID,
		IsDisabled:       isDisabled,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case err == actions.ErrCategoryNotFound:
			return nil, connect.NewError(connect.CodeNotFound, err)
		case err == actions.ErrParentCategoryNotFound:
			return nil, connect.NewError(connect.CodeNotFound, err)
		case err == actions.ErrSpecifiedCategoryParentIsNotParent:
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&budgetv1.UpdateCategoryResponse{}), nil
}
