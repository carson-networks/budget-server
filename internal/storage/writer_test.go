package storage

import (
	"context"
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage/category"
)

func TestNewWriterForTest_ReturnsWriterWithMocks(t *testing.T) {
	wt := NewWriterForTest()
	require.NotNil(t, wt.Account)
	require.NotNil(t, wt.Transaction)
	require.NotNil(t, wt.Category)
}

func TestMockICategoryWriter_Create_Update_StructParams(t *testing.T) {
	mockCat := &MockICategoryWriter{}
	catID := uuid.Must(uuid.FromString("550e8400-e29b-41d4-a716-446655440011"))
	mockCat.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(c *category.CategoryCreate) bool {
			return c != nil && c.Name == "Food" && !c.IsParent && !c.IsDisabled && c.CategoryType == category.CatergoryType_Income
		})).
		Return(nil)
	newName := "Food & Groceries"
	mockCat.EXPECT().
		Update(mock.Anything, catID, mock.MatchedBy(func(u *category.CategoryUpdate) bool {
			return u != nil && u.Name != nil && *u.Name == newName
		})).
		Return(nil)

	err := mockCat.Create(context.Background(), &category.CategoryCreate{
		Name:         "Food",
		IsParent:     false,
		IsDisabled:   false,
		CategoryType: category.CatergoryType_Income,
	})
	require.NoError(t, err)
	err = mockCat.Update(context.Background(), catID, &category.CategoryUpdate{Name: &newName})
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
}
