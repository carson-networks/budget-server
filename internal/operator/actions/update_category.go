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
	ErrCategoryNotFound                   = errors.New("category not found")
	ErrSpecifiedCategoryParentIsNotParent = errors.New("specified category parent is not a parent category")
)

type UpdateCategory struct {
	ID               uuid.UUID
	Name             *string
	ParentCategoryID *uuid.UUID
	IsDisabled       *bool

	IAction
}

func (u *UpdateCategory) Perform(ctx context.Context, writer *storage.Writer) error {
	existing, err := writer.Category.GetByID(ctx, u.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCategoryNotFound
		}
		return err
	}
	if existing == nil {
		return ErrCategoryNotFound
	}

	if u.ParentCategoryID != nil {
		parent, err := writer.Category.GetByID(ctx, *u.ParentCategoryID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrParentCategoryNotFound
			}
			return err
		}
		if !parent.IsParent {
			return ErrSpecifiedCategoryParentIsNotParent
		}
	}

	update := &category.CategoryUpdate{
		Name:             u.Name,
		ParentCategoryID: u.ParentCategoryID,
		IsDisabled:       u.IsDisabled,
	}
	return writer.Category.Update(ctx, u.ID, update)
}
