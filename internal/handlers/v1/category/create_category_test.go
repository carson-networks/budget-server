package category

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

func newCreateCategoryTestAPI(t *testing.T, op operator.IProcessor, reader categoryReader) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewCreateCategoryHandler(op, reader).Register(api)
	return api
}

func TestHTTP_CreateCategory_Success(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			cc, ok := a.(*actions.CreateCategory)
			return ok &&
				cc.Name == "Food" &&
				cc.IsParent == false &&
				cc.ParentCategoryID != nil &&
				*cc.ParentCategoryID == parentID &&
				cc.CategoryType == category.CatergoryType_Expense
		})).
		Return(nil)

	parentIDStr := parentID.String()
	resp := newCreateCategoryTestAPI(t, mockOp, &mockCategoryReader{}).Post("/v1/categories/create", CreateCategoryBody{
		Name:             "Food",
		IsParent:         false,
		ParentCategoryID: &parentIDStr,
		IsDisabled:       false,
		CategoryType:     1,
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateCategory_MustHaveParent(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrStandaloneCategoryNotSupported)

	resp := newCreateCategoryTestAPI(t, mockOp, &mockCategoryReader{}).Post("/v1/categories/create", CreateCategoryBody{
		Name:             "Leaf",
		IsParent:         false,
		ParentCategoryID: nil,
		CategoryType:     1,
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateCategory_ParentNotFound(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrParentCategoryNotFound)

	parentID := uuid.Must(uuid.NewV4()).String()
	resp := newCreateCategoryTestAPI(t, mockOp, &mockCategoryReader{}).Post("/v1/categories/create", CreateCategoryBody{
		Name:             "Leaf",
		IsParent:         false,
		ParentCategoryID: &parentID,
		CategoryType:     1,
	})

	assert.Equal(t, http.StatusNotFound, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateCategory_ParentMustBeParent(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrParentCategoryIsNotParent)

	parentID := uuid.Must(uuid.NewV4()).String()
	resp := newCreateCategoryTestAPI(t, mockOp, &mockCategoryReader{}).Post("/v1/categories/create", CreateCategoryBody{
		Name:             "Leaf",
		IsParent:         false,
		ParentCategoryID: &parentID,
		CategoryType:     1,
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	mockOp.AssertExpectations(t)
}
