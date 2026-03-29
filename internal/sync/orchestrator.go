package sync

import (
	"context"
	"fmt"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
	syncstore "github.com/carson-networks/budget-server/internal/storage/sync"
	"github.com/gofrs/uuid/v5"
)

type IStorage interface {
	Read() *storage.Reader
	Write(ctx context.Context) (*storage.Writer, error)
}

type Orchestrator struct {
	registry *Registry
	storage  IStorage
}

func NewOrchestrator(registry *Registry, storage IStorage) *Orchestrator {
	return &Orchestrator{
		registry: registry,
		storage:  storage,
	}
}

func (o *Orchestrator) Sync(ctx context.Context, accountIDs []uuid.UUID) error {
	reader := o.storage.Read()

	var syncs []*syncstore.Sync
	var err error
	if len(accountIDs) == 0 {
		syncs, err = reader.Sync.ListAll(ctx)
	} else {
		syncs, err = reader.Sync.ListByAccountIDs(ctx, accountIDs)
	}
	if err != nil {
		return fmt.Errorf("reading sync records: %w", err)
	}
	if len(syncs) == 0 {
		return nil
	}

	byType := make(map[syncstore.SyncType][]uuid.UUID)
	for _, s := range syncs {
		byType[s.SyncType] = append(byType[s.SyncType], s.AccountID)
	}

	for syncType, ids := range byType {
		provider, ok := o.registry.Get(syncType)
		if !ok {
			return fmt.Errorf("no provider registered for sync type %d", syncType)
		}

		actionsByAccount, err := provider.Sync(ctx, reader, ids)
		if err != nil {
			return fmt.Errorf("provider sync (type %d): %w", syncType, err)
		}

		for accountID, actionsToRun := range actionsByAccount {
			if err := o.executeActions(ctx, actionsToRun); err != nil {
				return fmt.Errorf("applying sync for account %s: %w", accountID, err)
			}
		}
	}

	return nil
}

func (o *Orchestrator) executeActions(ctx context.Context, actions []actions.IAction) error {
	writer, err := o.storage.Write(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	for _, action := range actions {
		if err := action.Perform(ctx, writer); err != nil {
			_ = writer.Rollback()
			return err
		}
	}

	return writer.Commit()
}
