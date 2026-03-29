package sync

import (
	"time"

	"github.com/carson-networks/budget-server/internal/storage/sqlconfig/bobgen"
	"github.com/gofrs/uuid/v5"
)

// SyncType identifies the sync provider responsible for an account.
// Values are explicit integers — never use iota for DB-persisted enums.
type SyncType int16

const (
	SyncType_Unknown SyncType = 0
	SyncType_Plaid   SyncType = 1
)

// Sync represents a row in the syncs table.
type Sync struct {
	AccountID uuid.UUID
	SyncType  SyncType
	CreatedAt time.Time
	UpdatedAt time.Time
}

func bobSyncToSync(row *bobgen.Sync) *Sync {
	return &Sync{
		AccountID: row.AccountID,
		SyncType:  SyncType(row.SyncType),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
