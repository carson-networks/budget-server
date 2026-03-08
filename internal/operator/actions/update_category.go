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
	ErrCategoryNotFound = errors.New("category not found")
)

type UpdateCategory struct {
	ID               uuid.UUID
	Name             *string
	ParentID         *uuid.UUID
	ShouldBeBudgeted *bool
	IsDisabled       *bool

	IAction
}

func (u *UpdateCategory) Perform(ctx context.Context, writer *storage.Writer) error {
	_, err := writer.Category.GetByID(ctx, u.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCategoryNotFound
		}
		return err
	}

	if u.ParentID != nil {
		parent, err := writer.Category.GetByID(ctx, *u.ParentID)
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

	update := &category.CategoryUpdate{
		Name:             u.Name,
		ParentID:         u.ParentID,
		ShouldBeBudgeted: u.ShouldBeBudgeted,
		IsDisabled:       u.IsDisabled,
	}
	return writer.Category.Update(ctx, u.ID, update)
}
