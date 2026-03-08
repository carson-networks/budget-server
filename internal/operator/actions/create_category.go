package actions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/carson-networks/budget-server/internal/storage/category"
	"github.com/gofrs/uuid/v5"
)

var (
	ErrCategoryMustBeInGroup  = errors.New("category must be in a group; parentID is required for non-group categories")
	ErrParentCategoryNotFound = errors.New("parent category not found")
	ErrParentMustBeGroup      = errors.New("parent must be a group")
)

type CreateCategory struct {
	Name             string
	IsGroup          bool
	ParentID         *uuid.UUID
	ShouldBeBudgeted bool
	IsDisabled       bool
	CategoryType     category.CategoryType

	IAction
}

func (c *CreateCategory) Perform(ctx context.Context, writer *storage.Writer) error {
	if !c.IsGroup && c.ParentID == nil {
		return ErrCategoryMustBeInGroup
	}
	if c.ParentID != nil {
		parent, err := writer.Category.GetByID(ctx, *c.ParentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrParentCategoryNotFound
			}
			return err
		}
		if !parent.IsGroup {
			return ErrParentMustBeGroup
		}
	}

	create := &category.CategoryCreate{
		Name:             c.Name,
		IsGroup:          c.IsGroup,
		ParentID:         c.ParentID,
		ShouldBeBudgeted: c.ShouldBeBudgeted,
		IsDisabled:       c.IsDisabled,
		CategoryType:     c.CategoryType,
	}
	_, err := writer.Category.Create(ctx, create)
	return err
}
