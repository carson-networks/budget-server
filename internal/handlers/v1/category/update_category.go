package category

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
)

// UpdateCategoryPath is the path parameters for updating a category.
type UpdateCategoryPath struct {
	ID string `path:"id" doc:"Category UUID"`
}

// UpdateCategoryBody is the request body for updating a category.
type UpdateCategoryBody struct {
	Name             *string `json:"name,omitempty" doc:"Category name"`
	ParentCategoryID *string `json:"parentCategoryID,omitempty" doc:"Parent category UUID"`
	IsDisabled       *bool   `json:"isDisabled,omitempty" doc:"Whether the category is disabled for new transactions"`
}

// UpdateCategoryInput is the Huma input for updating a category.
type UpdateCategoryInput struct {
	Path UpdateCategoryPath
	Body UpdateCategoryBody
}

// UpdateCategoryOutput is the Huma output for updating a category.
type UpdateCategoryOutput struct {
}

// UpdateCategoryHandler handles PATCH /v1/categories/{id}.
type UpdateCategoryHandler struct {
	Operator       operator.IProcessor
	CategoryReader categoryReader
}

// NewUpdateCategoryHandler creates a new UpdateCategoryHandler.
func NewUpdateCategoryHandler(op operator.IProcessor, reader categoryReader) *UpdateCategoryHandler {
	return &UpdateCategoryHandler{Operator: op, CategoryReader: reader}
}

// Register registers the update category endpoint with the Huma API.
func (h *UpdateCategoryHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "update-category",
		Method:      http.MethodPatch,
		Path:        "/v1/categories/update/{id}",
		Summary:     "Update category",
		Description: "Updates an existing category.",
		Tags:        []string{"Categories"},
	}, h.handle)
}

func (h *UpdateCategoryHandler) handle(ctx context.Context, input *UpdateCategoryInput) (*UpdateCategoryOutput, error) {
	id, err := uuid.FromString(input.Path.ID)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid category id", err)
	}

	var parentCategoryID *uuid.UUID
	if input.Body.ParentCategoryID != nil && *input.Body.ParentCategoryID != "" {
		pid, err := uuid.FromString(*input.Body.ParentCategoryID)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid parentCategoryID", err)
		}
		parentCategoryID = &pid
	}

	action := &actions.UpdateCategory{
		ID:               id,
		Name:             input.Body.Name,
		ParentCategoryID: parentCategoryID,
		IsDisabled:       input.Body.IsDisabled,
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		switch {
		case err == actions.ErrCategoryNotFound:
			return nil, huma.NewError(http.StatusNotFound, "category not found", err)
		case err == actions.ErrParentCategoryNotFound:
			return nil, huma.NewError(http.StatusNotFound, "parent category not found", err)
		case err == actions.ErrSpecifiedCategoryParentIsNotParent:
			return nil, huma.NewError(http.StatusBadRequest, "specified category parent is not a parent category", err)
		default:
			return nil, huma.NewError(http.StatusInternalServerError, "failed to update category", err)
		}
	}

	return &UpdateCategoryOutput{}, nil
}
