package sync

import (
	"context"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	"github.com/gofrs/uuid/v5"
)

type Provider interface {
	Type() syncstore.SyncType
	Sync(ctx context.Context, reader *storage.Reader, accountIDs []uuid.UUID) (map[uuid.UUID][]actions.IAction, error)
}
