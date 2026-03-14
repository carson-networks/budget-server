package category

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/storage/category"
)

type mockCategoryReader struct {
	mock.Mock
}

func (m *mockCategoryReader) List(ctx context.Context, filter *category.CategoryFilter) (*category.CategoryListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.CategoryListResult), args.Error(1)
}

func (m *mockCategoryReader) GetByID(ctx context.Context, id uuid.UUID) (*category.Category, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*category.Category), args.Error(1)
}

func newListCategoriesTestAPI(t *testing.T, reader categoryReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewListCategoriesHandler(reader).Register(api)
	return api
}

func TestHTTP_ListCategories_Empty(t *testing.T) {
	mockReader := &mockCategoryReader{}
	mockReader.On("List", mock.Anything, mock.Anything).
		Return(&category.CategoryListResult{Categories: []*category.Category{}, NextCursor: nil}, nil)

	resp := newListCategoriesTestAPI(t, mockReader).Get("/v1/categories")

	assert.Equal(t, http.StatusOK, resp.Code)
	mockReader.AssertExpectations(t)
}

func TestHTTP_ListCategories_NonEmpty(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	parentID := uuid.Must(uuid.NewV4())
	now := time.Now()
	mockReader := &mockCategoryReader{}
	mockReader.On("List", mock.Anything, mock.MatchedBy(func(f *category.CategoryFilter) bool {
		return f != nil && f.Limit == 20 && f.Offset == 0
	})).
		Return(&category.CategoryListResult{
			Categories: []*category.Category{
				{
					ID:               id,
					Name:             "Food",
					IsParent:         false,
					ParentCategoryID: &parentID,
					IsDisabled:       false,
					CategoryType:     category.CatergoryType_Expense,
					CreatedAt:        now,
				},
			},
			NextCursor: nil,
		}, nil)

	resp := newListCategoriesTestAPI(t, mockReader).Get("/v1/categories")

	assert.Equal(t, http.StatusOK, resp.Code)
	mockReader.AssertExpectations(t)
}
