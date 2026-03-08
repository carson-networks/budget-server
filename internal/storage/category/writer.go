package category

import (
	"context"
	"errors"

	"github.com/aarondl/opt/omit"
	"github.com/aarondl/opt/omitnull"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

type Writer struct {
	tx bob.Tx
	Reader
}

func NewWriter(tx bob.Tx) *Writer {
	return &Writer{
		tx: tx,
		Reader: Reader{
			exec: tx,
		},
	}
}

func (w *Writer) Create(ctx context.Context, create *CategoryCreate) (uuid.UUID, error) {
	setter := &bobgen.CategorySetter{
		Name:             omit.From(create.Name),
		IsGroup:          omit.From(create.IsGroup),
		ShouldBeBudgeted: omit.From(create.ShouldBeBudgeted),
		IsDisabled:       omit.From(create.IsDisabled),
		CategoryType:     omit.From(int16(create.CategoryType)),
	}
	if create.ParentID != nil {
		setter.ParentID = omitnull.From(*create.ParentID)
	} else if create.ParentID == nil && create.IsGroup == false {
		return uuid.Nil, errors.New("parentID must be set if IsGroup is false")
	}
	row, err := bobgen.Categories.Insert(setter).One(ctx, w.tx)
	if err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

func (w *Writer) Update(ctx context.Context, id uuid.UUID, update *CategoryUpdate) error {
	setter := bobgen.CategorySetter{}
	if update.Name != nil {
		setter.Name = omit.From(*update.Name)
	}
	if update.ParentID != nil {
		setter.ParentID = omitnull.From(*update.ParentID)
	}
	if update.ShouldBeBudgeted != nil {
		setter.ShouldBeBudgeted = omit.From(*update.ShouldBeBudgeted)
	}
	if update.IsDisabled != nil {
		setter.IsDisabled = omit.From(*update.IsDisabled)
	}
	if len(setter.SetColumns()) == 0 {
		return nil
	}
	_, err := bobgen.Categories.Update(
		setter.UpdateMod(),
		um.Where(bobgen.Categories.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, w.tx)
	return err
}
