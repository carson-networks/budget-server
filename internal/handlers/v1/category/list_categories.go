package category

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

// Category is the API response model for a category.
type Category struct {
	ID               string  `json:"id" doc:"Category UUID"`
	Name             string  `json:"name" doc:"Category name"`
	IsParent         bool    `json:"IsParent" doc:"Whether this category is a group"`
	ParentCategoryID *string `json:"ParentCategoryID,omitempty" doc:"Parent category UUID for non-root"`
	IsDisabled       bool    `json:"isDisabled" doc:"Whether the category is disabled for new transactions"`
	CategoryType     int     `json:"categoryType" doc:"Category direction: 0=Income, 1=Expense"`
	CreatedAt        string  `json:"createdAt" doc:"RFC3339 creation timestamp"`
}

// ListCategoriesInput is the Huma input for listing categories.
type ListCategoriesInput struct {
	Position int `query:"position" minimum:"0" doc:"Offset for pagination"`
	Limit    int `query:"limit" minimum:"1" maximum:"100" doc:"Page size, default 20"`
}

// ListCategoriesResponseBody is the response body for listing categories.
type ListCategoriesResponseBody struct {
	Categories []Category `json:"categories" doc:"Page of categories"`
	NextCursor *struct {
		Position int `json:"position" doc:"Offset for next page"`
		Limit    int `json:"limit" doc:"Page size"`
	} `json:"nextCursor,omitempty" doc:"Cursor to fetch the next page, absent on the last page"`
}

// ListCategoriesOutput is the Huma output for listing categories.
type ListCategoriesOutput struct {
	Body ListCategoriesResponseBody
}

type categoryReader interface {
	List(ctx context.Context, filter *category.CategoryFilter) (*category.CategoryListResult, error)
	GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error)
}

// ListCategoriesHandler handles GET /v1/categories.
type ListCategoriesHandler struct {
	CategoryReader categoryReader
}

// NewListCategoriesHandler creates a new ListCategoriesHandler.
func NewListCategoriesHandler(reader categoryReader) *ListCategoriesHandler {
	return &ListCategoriesHandler{CategoryReader: reader}
}

// Register registers the list categories endpoint with the Huma API.
func (h *ListCategoriesHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-categories",
		Method:      http.MethodGet,
		Path:        "/v1/categories",
		Summary:     "List categories",
		Description: "Returns a paginated list of categories.",
		Tags:        []string{"Categories"},
	}, h.handle)
}

func (h *ListCategoriesHandler) handle(ctx context.Context, input *ListCategoriesInput) (*ListCategoriesOutput, error) {
	logData := logging.GetLogData(ctx)

	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	filter := &category.CategoryFilter{
		Limit:  limit,
		Offset: input.Position,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listCategoriesMs")
	}
	result, err := h.CategoryReader.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list categories", err)
	}

	categories := result.Categories
	if categories == nil {
		categories = []*category.Category{}
	}

	if logData != nil {
		logData.AddData("categoryCount", len(categories))
	}

	resp := ListCategoriesResponseBody{
		Categories: make([]Category, len(categories)),
	}

	for i, cat := range categories {
		apiCat := Category{
			ID:           cat.ID.String(),
			Name:         cat.Name,
			IsParent:     cat.IsParent,
			IsDisabled:   cat.IsDisabled,
			CategoryType: int(cat.CategoryType),
			CreatedAt:    cat.CreatedAt.Format(time.RFC3339),
		}
		if cat.ParentCategoryID != nil {
			s := cat.ParentCategoryID.String()
			apiCat.ParentCategoryID = &s
		}
		resp.Categories[i] = apiCat
	}

	if result.NextCursor != nil {
		resp.NextCursor = &struct {
			Position int `json:"position" doc:"Offset for next page"`
			Limit    int `json:"limit" doc:"Page size"`
		}{
			Position: result.NextCursor.Position,
			Limit:    result.NextCursor.Limit,
		}
	}

	return &ListCategoriesOutput{Body: resp}, nil
}
