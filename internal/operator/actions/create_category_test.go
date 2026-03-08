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

func TestCreateCategory_Perform_Success_GroupWithNoParent(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(c *category.CategoryCreate) bool {
			return c != nil && c.Name == "Expenses" && c.IsGroup && c.ParentID == nil
		})).
		Return(catID, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Expenses",
		IsGroup:          true,
		ParentID:         nil,
		ShouldBeBudgeted: true,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_Success_LeafWithParent(t *testing.T) {
	parentID := uuid.Must(uuid.NewV4())
	catID := uuid.Must(uuid.NewV4())
	parent := &category.Category{
		ID: parentID, Name: "Expenses", IsGroup: true, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)
	mockCat.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(c *category.CategoryCreate) bool {
			return c != nil && c.Name == "Food" && !c.IsGroup && c.ParentID != nil && *c.ParentID == parentID
		})).
		Return(catID, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Food",
		IsGroup:          false,
		ParentID:         &parentID,
		ShouldBeBudgeted: true,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestCreateCategory_Perform_MissingParentIDForNonGroup(t *testing.T) {
	wt := storage.NewWriterForTest()
	action := &CreateCategory{
		Name:             "Food",
		IsGroup:          false,
		ParentID:         nil,
		ShouldBeBudgeted: true,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryMustBeInGroup)
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
		IsGroup:          false,
		ParentID:         &parentID,
		ShouldBeBudgeted: true,
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
		ID: parentID, Name: "Leaf", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Sub",
		IsGroup:          false,
		ParentID:         &parentID,
		ShouldBeBudgeted: true,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParentMustBeGroup)
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
		IsGroup:          false,
		ParentID:         &parentID,
		ShouldBeBudgeted: true,
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
		Return(uuid.Nil, createErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &CreateCategory{
		Name:             "Expenses",
		IsGroup:          true,
		ParentID:         nil,
		ShouldBeBudgeted: true,
		IsDisabled:       false,
		CategoryType:     category.CatergoryType_Expense,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, createErr)
	mockCat.AssertExpectations(t)
}
