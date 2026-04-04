package v1Category

import (
	"context"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/category"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	storagecategory "github.com/carson-networks/budget-server/internal/storage/category"
	"github.com/gofrs/uuid/v5"
)

// CreateCategory implements budget.v1.CategoryService.CreateCategory.
func (s *Service) CreateCategory(ctx context.Context, req *connect.Request[category.CreateCategoryRequest]) (*connect.Response[category.CreateCategoryResponse], error) {
	var parentCategoryID *uuid.UUID
	if req.Msg.ParentCategoryId != nil && *req.Msg.ParentCategoryId != "" {
		id, err := uuid.FromString(*req.Msg.ParentCategoryId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		parentCategoryID = &id
	}

	action := &actions.CreateCategory{
		Name:             req.Msg.GetName(),
		IsParent:         req.Msg.GetIsParent(),
		ParentCategoryID: parentCategoryID,
		IsDisabled:       req.Msg.GetIsDisabled(),
		CategoryType:     storagecategory.CategoryType(req.Msg.GetCategoryType()),
	}

	if err := s.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrStandaloneCategoryNotSupported):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, actions.ErrParentCategoryNotFound):
			return nil, connect.NewError(connect.CodeNotFound, err)
		case errors.Is(err, actions.ErrParentCategoryIsNotParent):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		default:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	}
	return connect.NewResponse(&category.CreateCategoryResponse{Status: http.StatusCreated}), nil
}
