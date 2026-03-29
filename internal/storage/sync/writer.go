package sync

import (
	"context"

	"github.com/aarondl/opt/omit"
	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
	"github.com/stephenafamo/bob"
)

type Writer struct {
	tx bob.Tx
	Reader
}

func NewWriter(tx bob.Tx) *Writer {
	return &Writer{
		tx:     tx,
		Reader: Reader{exec: tx},
	}
}

func (w *Writer) Create(ctx context.Context, accountID uuid.UUID, syncType SyncType) error {
	setter := &bobgen.SyncSetter{
		AccountID: omit.From(accountID),
		SyncType:  omit.From(int16(syncType)),
	}
	_, err := bobgen.Syncs.Insert(setter).One(ctx, w.tx)
	return err
}
func (w *Writer) Delete(ctx context.Context, accountID uuid.UUID) error {
	_, err := bobgen.Syncs.Delete(
		bobgen.DeleteWhere.Syncs.AccountID.EQ(accountID),
	).Exec(ctx, w.tx)
	return err
}
