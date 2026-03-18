package actions

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/budget"
	"github.com/carson-networks/budget-server/internal/storage/category"
)

func validLeafCategory(id uuid.UUID) *category.Category {
	return &category.Category{
		ID: id, Name: "Food", IsParent: false, IsDisabled: false,
		CategoryType: category.CatergoryType_Expense,
	}
}

func TestSetBudget_Perform_Success(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromInt(500)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validLeafCategory(categoryID), nil)

	mockBudget := &storage.MockIBudgetWriter{}
	mockBudget.EXPECT().
		Set(mock.Anything, mock.MatchedBy(func(s *budget.BudgetSet) bool {
			return s.CategoryID == categoryID && s.Month.Equal(month) && s.Amount.Equal(amount)
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Budget = mockBudget

	action := &SetBudget{
		CategoryID:            categoryID,
		Month:                 month,
		Amount:                amount,
		OverwriteFutureMonths: false,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
	mockBudget.AssertExpectations(t)
}

func TestSetBudget_Perform_WithOverwriteFutureMonths(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromInt(500)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validLeafCategory(categoryID), nil)

	mockBudget := &storage.MockIBudgetWriter{}
	mockBudget.EXPECT().
		DeleteByCategoryAndMonthsAfter(mock.Anything, categoryID, month).
		Return(nil)
	mockBudget.EXPECT().
		Set(mock.Anything, mock.MatchedBy(func(s *budget.BudgetSet) bool {
			return s.CategoryID == categoryID && s.Month.Equal(month) && s.Amount.Equal(amount)
		})).
		Return(nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Budget = mockBudget

	action := &SetBudget{
		CategoryID:            categoryID,
		Month:                 month,
		Amount:                amount,
		OverwriteFutureMonths: true,
	}

	err := action.Perform(context.Background(), wt)
	require.NoError(t, err)
	mockCat.AssertExpectations(t)
	mockBudget.AssertExpectations(t)
}

func TestSetBudget_Perform_CategoryNotFound(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(nil, sql.ErrNoRows)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat

	action := &SetBudget{
		CategoryID: categoryID,
		Month:      month,
		Amount:     decimal.NewFromInt(500),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryNotFoundForBudget)
	mockCat.AssertExpectations(t)
}

func TestSetBudget_Perform_CategoryIsParent(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	cat := validLeafCategory(categoryID)
	cat.IsParent = true

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(cat, nil)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat

	action := &SetBudget{
		CategoryID: categoryID,
		Month:      month,
		Amount:     decimal.NewFromInt(500),
	}

	err := action.Perform(context.Background(), wt)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCategoryIsParent)
	mockCat.AssertExpectations(t)
}

func TestSetBudget_Perform_SetError(t *testing.T) {
	setErr := errors.New("set failed")
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	amount := decimal.NewFromInt(500)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validLeafCategory(categoryID), nil)

	mockBudget := &storage.MockIBudgetWriter{}
	mockBudget.EXPECT().
		Set(mock.Anything, mock.Anything).
		Return(setErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Budget = mockBudget

	action := &SetBudget{
		CategoryID: categoryID,
		Month:      month,
		Amount:     amount,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, setErr)
	mockCat.AssertExpectations(t)
	mockBudget.AssertExpectations(t)
}

func TestSetBudget_Perform_DeleteByCategoryAndMonthsAfterError(t *testing.T) {
	deleteErr := errors.New("delete failed")
	categoryID := uuid.Must(uuid.NewV4())
	month := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	mockCat := &storage.MockICategoryWriter{}
	mockCat.EXPECT().
		GetByID(mock.Anything, categoryID).
		Return(validLeafCategory(categoryID), nil)

	mockBudget := &storage.MockIBudgetWriter{}
	mockBudget.EXPECT().
		DeleteByCategoryAndMonthsAfter(mock.Anything, categoryID, month).
		Return(deleteErr)

	wt := storage.NewWriterForTest()
	wt.Category = mockCat
	wt.Budget = mockBudget

	action := &SetBudget{
		CategoryID:            categoryID,
		Month:                 month,
		Amount:                decimal.NewFromInt(500),
		OverwriteFutureMonths: true,
	}

	err := action.Perform(context.Background(), wt)
	assert.ErrorIs(t, err, deleteErr)
	mockCat.AssertExpectations(t)
	mockBudget.AssertExpectations(t)
}
