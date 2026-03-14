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
	IsParent         bool
	ParentCategoryID *uuid.UUID
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

// CategoryCursor identifies a position in a paginated result set.
type CategoryCursor struct {
	Position int
	Limit    int
}

// CategoryListResult contains a page of categories and an optional next cursor.
type CategoryListResult struct {
	Categories []*Category
	NextCursor *CategoryCursor
}

// CategoryCreate is the input for creating a category.
type CategoryCreate struct {
	Name             string
	IsParent         bool
	ParentCategoryID *uuid.UUID // required when IsParent is false; nil for root groups
	IsDisabled       bool
	CategoryType     CategoryType
}

// CategoryUpdate is the input for updating a category (mutable fields only).
type CategoryUpdate struct {
	Name             *string
	ParentCategoryID *uuid.UUID
	IsDisabled       *bool
	CategoryType     *CategoryType
}

func bobCategoryToCategory(row *bobgen.Category) *Category {
	var parentCategoryID *uuid.UUID
	if row.ParentID.IsValue() {
		id := row.ParentID.MustGet()
		parentCategoryID = &id
	}
	return &Category{
		ID:               row.ID,
		Name:             row.Name,
		IsParent:         row.IsGroup,
		ParentCategoryID: parentCategoryID,
		IsDisabled:       row.IsDisabled,
		CategoryType:     CategoryType(row.CategoryType),
		CreatedAt:        row.CreatedAt,
	}
}
