package category

import (
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
)

// CategoryType represents the direction of the category (e.g. expense vs income).
// Stored as SMALLINT (category_type) in the database.
type CategoryType int16

// Category represents a category record.
type Category struct {
	ID               uuid.UUID
	Name             string
	IsGroup          bool
	ParentID         *uuid.UUID // nil for root groups
	ParentName       string     // optional, populated when joining parent
	ShouldBeBudgeted bool
	IsDisabled       bool
	CategoryType     CategoryType
	CreatedAt        time.Time
}

// CategoryFilter specifies filters for listing categories.
type CategoryFilter struct {
	Limit      int
	Offset     int
	ParentID   *uuid.UUID // optional: filter by parent
	IsDisabled *bool      // optional: filter by disabled state
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
	ParentID         *uuid.UUID // set to this UUID
	ClearParentID    bool       // if true, set parent_id to NULL (e.g. make group a root)
	ShouldBeBudgeted *bool
	IsDisabled       *bool
}

func bobCategoryToCategory(row *bobgen.Category, parentName string) *Category {
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
		ParentName:       parentName,
		ShouldBeBudgeted: row.ShouldBeBudgeted,
		IsDisabled:       row.IsDisabled,
		CategoryType:     CategoryType(row.CategoryType),
		CreatedAt:        row.CreatedAt,
	}
}
