package v1Category

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect/v2"

	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
)

// CreateCategory implements category.v1.CategoryService.CreateCategory.
func (s *Service) CreateCategory(ctx context.Context, req *category.CreateCategoryRequest) (*category.CreateCategoryResponse, error) {
	var parentCategoryID *uuid.UUID
	if req.ParentCategoryId != nil && *req.ParentCategoryId != "" {
		id, err := uuid.FromString(*req.ParentCategoryId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		}
		parentCategoryID = &id
	}

	catType, err := FromConnectCategoryType(req.GetCategoryType())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
	}

	action := &actions.CreateCategory{
		Name:             req.GetName(),
		IsParent:         req.GetIsParent(),
		ParentCategoryID: parentCategoryID,
		IsDisabled:       req.GetIsDisabled(),
		CategoryType:     catType,
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrStandaloneCategoryNotSupported):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrParentCategoryNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err.Error()).WithCause(err)
		case errors.Is(err, actions.ErrParentCategoryIsNotParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err.Error()).WithCause(err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err.Error()).WithCause(err)
		}
	}
	return &category.CreateCategoryResponse{Status: http.StatusCreated}, nil
}
