package category

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/mods"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

func (r *Reader) List(ctx context.Context, filter *CategoryFilter) ([]*Category, error) {
	limit := 20
	offset := 0
	if filter != nil {
		if filter.Limit > 0 {
			limit = filter.Limit
		}
		offset = filter.Offset
	}

	var queryMods []bob.Mod[*dialect.SelectQuery]
	if filter != nil {
		var whereMods []mods.Where[*dialect.SelectQuery]
		if filter.ParentID != nil {
			whereMods = append(whereMods, bobgen.SelectWhere.Categories.ParentID.EQ(*filter.ParentID))
		}
		if filter.IsDisabled != nil {
			whereMods = append(whereMods, bobgen.SelectWhere.Categories.IsDisabled.EQ(*filter.IsDisabled))
		}
		if len(whereMods) == 1 {
			queryMods = append(queryMods, whereMods[0])
		} else if len(whereMods) > 1 {
			queryMods = append(queryMods, psql.WhereAnd(whereMods...))
		}
	}
	queryMods = append(queryMods,
		sm.Limit(limit+1),
		sm.Offset(offset),
		sm.OrderBy(bobgen.Categories.Columns.Name).Asc(),
		sm.OrderBy(bobgen.Categories.Columns.ID).Asc(),
	)

	rows, err := bobgen.Categories.Query(queryMods...).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	// Load parent names for parent_id display
	if err := rows.LoadParent(ctx, r.exec); err != nil {
		return nil, err
	}

	result := make([]*Category, len(rows))
	for i, row := range rows {
		parentName := ""
		if row.R.Parent != nil {
			parentName = row.R.Parent.Name
		}
		result[i] = bobCategoryToCategory(row, parentName)
	}
	return result, nil
}

func (r *Reader) GetByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	row, err := bobgen.FindCategory(ctx, r.exec, id)
	if err != nil {
		return nil, err
	}
	parentName := ""
	if row.ParentID.IsValue() {
		parent, err := row.Parent().One(ctx, r.exec)
		if err == nil && parent != nil {
			parentName = parent.Name
		}
	}
	return bobCategoryToCategory(row, parentName), nil
}
