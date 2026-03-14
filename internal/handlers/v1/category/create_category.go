package category

import (
	"context"
	"errors"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

// CreateCategoryBody is the request body for creating a category.
type CreateCategoryBody struct {
	Name             string  `json:"name" required:"true" doc:"Category name"`
	IsParent         bool    `json:"isParent" doc:"Whether this category is a parent"`
	ParentCategoryID *string `json:"parentCategoryID,omitempty" doc:"Parent category UUID; required when isParent is false"`
	IsDisabled       bool    `json:"isDisabled" doc:"Whether the category is disabled for new transactions"`
	CategoryType     int     `json:"categoryType" doc:"Category direction: 0=Income, 1=Expense"`
}

// CreateCategoryInput is the Huma input for creating a category.
type CreateCategoryInput struct {
	Body CreateCategoryBody
}

// CreateCategoryOutput is the Huma output for creating a category.
type CreateCategoryOutput struct {
	Status int `json:"status" doc:"HTTP status"`
}

// CreateCategoryHandler handles POST /v1/categories.
type CreateCategoryHandler struct {
	Operator       operator.IProcessor
	CategoryReader categoryReader
}

// NewCreateCategoryHandler creates a new CreateCategoryHandler.
func NewCreateCategoryHandler(op operator.IProcessor, reader categoryReader) *CreateCategoryHandler {
	return &CreateCategoryHandler{Operator: op, CategoryReader: reader}
}

// Register registers the create category endpoint with the Huma API.
func (h *CreateCategoryHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-category",
		Method:      http.MethodPost,
		Path:        "/v1/categories/create",
		Summary:     "Create category",
		Description: "Creates a new category.",
		Tags:        []string{"Categories"},
	}, h.handle)
}

func (h *CreateCategoryHandler) handle(ctx context.Context, input *CreateCategoryInput) (*CreateCategoryOutput, error) {
	var parentCatergoryID *uuid.UUID
	if input.Body.ParentCategoryID != nil && *input.Body.ParentCategoryID != "" {
		id, err := uuid.FromString(*input.Body.ParentCategoryID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid ParentCategoryID", err)
		}
		parentCatergoryID = &id
	}

	action := &actions.CreateCategory{
		Name:             input.Body.Name,
		IsParent:         input.Body.IsParent,
		ParentCategoryID: parentCatergoryID,
		IsDisabled:       input.Body.IsDisabled,
		CategoryType:     category.CategoryType(input.Body.CategoryType),
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		switch {
		case errors.Is(err, actions.ErrStandaloneCategoryNotSupported):
			return nil, huma.NewError(http.StatusBadRequest, "category must have parent; parentCategoryID is required for non-parent categories", err)
		case errors.Is(err, actions.ErrParentCategoryNotFound):
			return nil, huma.NewError(http.StatusNotFound, "parent category not found", err)
		case errors.Is(err, actions.ErrParentCategoryIsNotParent):
			return nil, huma.NewError(http.StatusBadRequest, "parent must be a parent category", err)
		default:
			return nil, huma.NewError(http.StatusInternalServerError, "failed to create category", err)
		}
	}

	return &CreateCategoryOutput{Status: http.StatusCreated}, nil
}
