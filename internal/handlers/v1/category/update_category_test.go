package category

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

func TestUpdateCategoryHandler_InvalidID(t *testing.T) {
	h := NewUpdateCategoryHandler(&operator.MockIProcessor{}, &mockCategoryReader{})
	out, err := h.handle(context.Background(), &UpdateCategoryInput{
		Path: UpdateCategoryPath{ID: "not-a-uuid"},
		Body: UpdateCategoryBody{},
	})
	assert.Nil(t, out)
	assert.NotNil(t, err)
	var statusErr huma.StatusError
	assert.True(t, errors.As(err, &statusErr))
	assert.Equal(t, http.StatusBadRequest, statusErr.GetStatus())
}

func TestUpdateCategoryHandler_Success(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	newName := "Food Updated"
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			uc, ok := a.(*actions.UpdateCategory)
			return ok && uc.ID == id && uc.Name != nil && *uc.Name == newName
		})).
		Return(nil)

	mockReader := &mockCategoryReader{}
	mockReader.On("GetByID", mock.Anything, id).
		Return(&category.Category{
			ID:           id,
			Name:         newName,
			IsParent:     false,
			CategoryType: category.CatergoryType_Expense,
		}, nil)

	h := NewUpdateCategoryHandler(mockOp, mockReader)
	out, err := h.handle(context.Background(), &UpdateCategoryInput{
		Path: UpdateCategoryPath{ID: id.String()},
		Body: UpdateCategoryBody{Name: &newName},
	})
	assert.NoError(t, err)
	assert.NotNil(t, out)
	mockOp.AssertExpectations(t)
	mockReader.AssertExpectations(t)
}

func TestUpdateCategoryHandler_NotFound(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrCategoryNotFound)

	id := uuid.Must(uuid.NewV4())
	h := NewUpdateCategoryHandler(mockOp, &mockCategoryReader{})
	out, err := h.handle(context.Background(), &UpdateCategoryInput{
		Path: UpdateCategoryPath{ID: id.String()},
		Body: UpdateCategoryBody{Name: ptrString("New Name")},
	})
	assert.Nil(t, out)
	assert.NotNil(t, err)
	var statusErr huma.StatusError
	assert.True(t, errors.As(err, &statusErr))
	assert.Equal(t, http.StatusNotFound, statusErr.GetStatus())
	mockOp.AssertExpectations(t)
}

func TestUpdateCategoryHandler_ParentMustBeParent(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(actions.ErrSpecifiedCategoryParentIsNotParent)

	id := uuid.Must(uuid.NewV4())
	parentID := uuid.Must(uuid.NewV4()).String()
	h := NewUpdateCategoryHandler(mockOp, &mockCategoryReader{})
	out, err := h.handle(context.Background(), &UpdateCategoryInput{
		Path: UpdateCategoryPath{ID: id.String()},
		Body: UpdateCategoryBody{ParentCategoryID: &parentID},
	})
	assert.Nil(t, out)
	assert.NotNil(t, err)
	var statusErr huma.StatusError
	assert.True(t, errors.As(err, &statusErr))
	assert.Equal(t, http.StatusBadRequest, statusErr.GetStatus())
	mockOp.AssertExpectations(t)
}

func ptrString(s string) *string {
	return &s
}
