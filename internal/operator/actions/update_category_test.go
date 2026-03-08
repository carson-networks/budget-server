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

func TestUpdateCategory_Perform_Success(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	newName := "Food & Groceries"
	existing := &category.Category{
		ID: catID, Name: "Food", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(existing, nil)
	mockCat.EXPECT().
		Update(mock.Anything, catID, mock.MatchedBy(func(u *category.CategoryUpdate) bool {
			return u != nil && u.Name != nil && *u.Name == newName
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &UpdateCategory{
		ID:   catID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestUpdateCategory_Perform_Success_WithNewParent(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	parentID := uuid.Must(uuid.NewV4())
	existing := &category.Category{
		ID: catID, Name: "Food", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}
	parent := &category.Category{
		ID: parentID, Name: "Expenses", IsGroup: true, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(existing, nil)
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)
	mockCat.EXPECT().
		Update(mock.Anything, catID, mock.MatchedBy(func(u *category.CategoryUpdate) bool {
			return u != nil && u.ParentID != nil && *u.ParentID == parentID
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &UpdateCategory{
		ID:       catID,
		ParentID: &parentID,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}

func TestUpdateCategory_Perform_CategoryNotFound(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	newName := "Updated"
	action := &UpdateCategory{
		ID:   catID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryNotFound)
	mockCat.AssertExpectations(t)
}

func TestUpdateCategory_Perform_ParentNotFound(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	parentID := uuid.Must(uuid.NewV4())
	existing := &category.Category{
		ID: catID, Name: "Food", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(existing, nil)
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &UpdateCategory{
		ID:       catID,
		ParentID: &parentID,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParentCategoryNotFound)
	mockCat.AssertExpectations(t)
}

func TestUpdateCategory_Perform_ParentNotAGroup(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	parentID := uuid.Must(uuid.NewV4())
	existing := &category.Category{
		ID: catID, Name: "Food", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}
	parent := &category.Category{
		ID: parentID, Name: "Leaf", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(existing, nil)
	mockCat.EXPECT().
		GetByID(mock.Anything, parentID).
		Return(parent, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	action := &UpdateCategory{
		ID:       catID,
		ParentID: &parentID,
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrParentMustBeGroup)
	mockCat.AssertExpectations(t)
}

func TestUpdateCategory_Perform_UpdateError(t *testing.T) {
	catID := uuid.Must(uuid.NewV4())
	updateErr := errors.New("update failed")
	existing := &category.Category{
		ID: catID, Name: "Food", IsGroup: false, ParentID: nil,
		ShouldBeBudgeted: true, IsDisabled: false, CategoryType: category.CatergoryType_Expense,
	}

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, catID).
		Return(existing, nil)
	mockCat.EXPECT().
		Update(mock.Anything, catID, mock.Anything).
		Return(updateErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	newName := "Updated"
	action := &UpdateCategory{
		ID:   catID,
		Name: &newName,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, updateErr)
	mockCat.AssertExpectations(t)
}
