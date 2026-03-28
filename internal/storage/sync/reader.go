package sync

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
)

type Reader struct {
	exec bob.Executor
}

func NewReader(exec bob.Executor) *Reader {
	return &Reader{exec: exec}
}

// ListAll returns all sync records.
func (r *Reader) ListAll(ctx context.Context) ([]*Sync, error) {
	rows, err := bobgen.Syncs.Query().All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertSyncs(rows), nil
}

// ListByAccountIDs returns sync records for the given account IDs.
func (r *Reader) ListByAccountIDs(ctx context.Context, accountIDs []uuid.UUID) ([]*Sync, error) {
	if len(accountIDs) == 0 {
		return nil, nil
	}
	rows, err := bobgen.Syncs.Query(
		bobgen.SelectWhere.Syncs.AccountID.In(accountIDs...),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertSyncs(rows), nil
}

// ListByType returns all sync records of the given type.
func (r *Reader) ListByType(ctx context.Context, syncType SyncType) ([]*Sync, error) {
	rows, err := bobgen.Syncs.Query(
		bobgen.SelectWhere.Syncs.SyncType.EQ(int16(syncType)),
	).All(ctx, r.exec)
	if err != nil {
		return nil, err
	}
	return convertSyncs(rows), nil
}

func convertSyncs(rows bobgen.SyncSlice) []*Sync {
	out := make([]*Sync, len(rows))
	for i, row := range rows {
		out[i] = bobSyncToSync(row)
	}
	return out
}
