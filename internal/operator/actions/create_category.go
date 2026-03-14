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
	ErrParentCategoryIsNotParent      = errors.New("parent category is not parent")
	ErrParentCategoryNotFound         = errors.New("parent category not found")
	ErrStandaloneCategoryNotSupported = errors.New("standalone category is not supported")
)

type CreateCategory struct {
	Name             string
	IsParent         bool
	ParentCategoryID *uuid.UUID
	IsDisabled       bool
	CategoryType     category.CategoryType

	IAction
}

func (c *CreateCategory) Perform(ctx context.Context, writer *storage.Writer) error {
	if !c.IsParent && c.ParentCategoryID == nil {
		return ErrStandaloneCategoryNotSupported
	}
	if c.ParentCategoryID != nil {
		parent, err := writer.Category.GetByID(ctx, *c.ParentCategoryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrParentCategoryNotFound
			}
			return err
		}
		if !parent.IsParent {
			return ErrParentCategoryIsNotParent
		}
	}

	create := &category.CategoryCreate{
		Name:             c.Name,
		IsParent:         c.IsParent,
		ParentCategoryID: c.ParentCategoryID,
		IsDisabled:       c.IsDisabled,
		CategoryType:     c.CategoryType,
	}
	err := writer.Category.Create(ctx, create)
	if err != nil {
		return err
	}
	return nil
}
