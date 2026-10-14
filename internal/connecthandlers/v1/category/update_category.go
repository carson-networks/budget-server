package v1Category

import (
	"context"
	"errors"

	"connectrpc.com/connect/v2"

	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
)

// UpdateCategory implements category.v1.CategoryService.UpdateCategory.
func (s *Service) UpdateCategory(ctx context.Context, req *category.UpdateCategoryRequest) (*category.UpdateCategoryResponse, error) {
	id, err := uuid.FromString(req.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	var name *string
	if req.Name != nil {
		name = req.Name
	}
	var parentCategoryID *uuid.UUID
	if req.ParentCategoryId != nil && *req.ParentCategoryId != "" {
		pid, perr := uuid.FromString(*req.ParentCategoryId)
		if perr != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, perr.Error()).WithCause(perr)
		}
		parentCategoryID = &pid
	}
	var isDisabled *bool
	if req.IsDisabled != nil {
		isDisabled = req.IsDisabled
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
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrParentCategoryNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrSpecifiedCategoryParentIsNotParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &category.UpdateCategoryResponse{}, nil
}
