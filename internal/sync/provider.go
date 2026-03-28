package sync

import (
	"context"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	"github.com/gofrs/uuid/v5"
)

// Provider handles the full sync lifecycle for one SyncType.
// The provider fetches external data, resolves what changed, builds
// provider-specific actions, and returns them grouped by account ID.
type Provider interface {
	// Type returns which SyncType this provider handles.
	Type() syncstore.SyncType

	// Sync fetches external data, resolves changes, builds actions, and
	// returns them grouped by account ID. Each action handles both core
	// table writes (transaction insert/update/delete, balance adjustment)
	// AND provider-specific writes (link tables, cursors, etc).
	//
	// Called by the orchestrator. The orchestrator will then execute the
	// actions per account, each in its own DB transaction.
	Sync(ctx context.Context, reader *storage.Reader, accountIDs []uuid.UUID) (map[uuid.UUID][]actions.IAction, error)
}
