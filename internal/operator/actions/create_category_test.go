package actions

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

func TestCreateCategory_Perform_Success(t *testing.T) {
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(c *category.CategoryCreate) bool {
			return c != nil && c.Name == "Expenses" && c.IsParent && c.ParentCategoryID == nil
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:         "Expenses",
		IsParent:     true,
		IsDisabled:   false,
		CategoryType: category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_Success_LeafWithParent(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	parent := &category.Category{
		ID: parentID, Name: "Expenses", IsParent: true, ParentCategoryID: nil, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)
	mockCat.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(c *category.CategoryCreate) bool {
			return c != nil && c.Name == "Food" && !c.IsParent && c.ParentCategoryID != nil && *c.ParentCategoryID == parentID
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Food",
		IsParent:         false,
		ParentCategoryID: &parentID,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_MissingParentIDForNonParent(t *testing.T) {
	wt := storage.NewWriterForTest()
	action := &CreateCategory{
		Name:             "Food",
		IsParent:         false,
		ParentCategoryID: nil,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrStandaloneCategoryNotSupported)
}

func TestCreateCategory_Perform_ParentNotFound(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Food",
		IsParent:         false,
		ParentCategoryID: &parentID,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParentCategoryNotFound)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_ParentNotAGroup(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	parent := &category.Category{
		ID: parentID, Name: "Leaf", IsParent: false, ParentCategoryID: nil,
		IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Sub",
		IsParent:         false,
		ParentCategoryID: &parentID,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParentCategoryIsNotParent)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_GetByIDError(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	dbErr := errors.New("db error")
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(nil, dbErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Food",
		IsParent:         false,
		ParentCategoryID: &parentID,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, dbErr)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_CreateError(t *testing.T) {
	createErr := errors.New("create failed")
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		Create(mock.Anything, mock.Anything).
		Return(createErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Expenses",
		IsParent:         true,
		ParentCategoryID: nil,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, createErr)
	mockCat.AssertExpectations(t)
}
