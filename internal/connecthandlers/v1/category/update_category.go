package v1Category

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
)

// UpdateCategory implements category.v1.CategoryService.UpdateCategory.
func (s *Service) UpdateCategory(ctx context.Context, req *connect.Request[category.UpdateCategoryRequest]) (*connect.Response[category.UpdateCategoryResponse], error) {
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
		case errors.Is(err, actions.ErrCategoryNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrParentCategoryNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrSpecifiedCategoryParentIsNotParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&category.UpdateCategoryResponse{}), nil
}
