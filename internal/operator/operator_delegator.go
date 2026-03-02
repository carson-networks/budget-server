package operator

import (
	"context"
	"sync"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
)

// OperatorDelegator manages the queue, starts/stops Operators (workers), and enqueues items.
type OperatorDelegator struct {
	storage    *storage.Storage
	queue      chan ActionItem
	numWorkers int
	wg         sync.WaitGroup
	stopOnce   sync.Once
}

func NewOperatorDelegator(s *storage.Storage, numWorkers int) *OperatorDelegator {
	if numWorkers < 1 {
		numWorkers = 1
	}
	return &OperatorDelegator{
		storage:    s,
		queue:      make(chan ActionItem, 1000),
		numWorkers: numWorkers,
	}
}

func (d *OperatorDelegator) Start() {
	for i := 0; i < d.numWorkers; i++ {
		d.wg.Add(1)
		op := NewOperator(d.storage, d.queue)
		go func() {
			defer d.wg.Done()
			op.Run()
		}()
	}
}

func (d *OperatorDelegator) Stop() {
	d.stopOnce.Do(func() {
		close(d.queue)
		d.wg.Wait()
	})
}

func (d *OperatorDelegator) Process(ctx context.Context, action actions.IAction) error {
	respCh := make(chan ActionItemResponse, 1)
	item := ActionItem{
		ctx:      ctx,
		action:   action,
		response: respCh,
	}

	d.queue <- item

	select {
	case resp := <-respCh:
		return resp.err
	case <-ctx.Done():
		return ctx.Err()
	}
}
