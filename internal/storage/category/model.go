package category

import (
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
)

type CategoryType int16

const (
	CatergoryType_Income = iota
	CatergoryType_Expense
)

// Category represents a category record.
type Category struct {
	ID               uuid.UUID
	Name             string
	IsGroup          bool
	ParentID         *uuid.UUID
	ShouldBeBudgeted bool
	IsDisabled       bool
	CategoryType     CategoryType
	CreatedAt        time.Time
}

// CategoryFilter specifies filters for listing categories.
type CategoryFilter struct {
	Limit      int
	Offset     int
	ParentID   *uuid.UUID
	IsDisabled *bool
}

// CategoryCreate is the input for creating a category.
type CategoryCreate struct {
	Name             string
	IsGroup          bool
	ParentID         *uuid.UUID // required when IsGroup is false; nil for root groups
	ShouldBeBudgeted bool
	IsDisabled       bool
	CategoryType     CategoryType
}

// CategoryUpdate is the input for updating a category (mutable fields only).
type CategoryUpdate struct {
	Name             *string
	ParentID         *uuid.UUID
	ShouldBeBudgeted *bool
	IsDisabled       *bool
}

func bobCategoryToCategory(row *bobgen.Category) *Category {
	var parentID *uuid.UUID
	if row.ParentID.IsValue() {
		id := row.ParentID.MustGet()
		parentID = &id
	}
	return &Category{
		ID:               row.ID,
		Name:             row.Name,
		IsGroup:          row.IsGroup,
		ParentID:         parentID,
		ShouldBeBudgeted: row.ShouldBeBudgeted,
		IsDisabled:       row.IsDisabled,
		CategoryType:     CategoryType(row.CategoryType),
		CreatedAt:        row.CreatedAt,
	}
}
