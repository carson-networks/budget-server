package operator

import (
	"context"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage"
)

// Operator is the worker that processes items from the queue.
type Operator struct {
	storage *storage.Storage
	queue   chan ActionItem
}

func NewOperator(s *storage.Storage, queue chan ActionItem) *Operator {
	return &Operator{
		storage: s,
		queue:   queue,
	}
}

// Run listens to the queue and processes items. Exits when the queue is closed.
func (o *Operator) Run() {
	for item := range o.queue {
		o.processItem(item)
	}
}

func (o *Operator) processItem(item ActionItem) {
	writer, err := o.storage.Write(item.ctx)
	if err != nil {
		item.response <- ActionItemResponse{err: err}
		return
	}

	err = item.action.Perform(item.ctx, writer)
	if err != nil {
		_ = writer.Rollback()
		item.response <- ActionItemResponse{err: err}
		return
	}

	if err = writer.Commit(); err != nil {
		item.response <- ActionItemResponse{err: err}
		return
	}

	item.response <- ActionItemResponse{}
}

type ActionItem struct {
	ctx      context.Context
	action   actions.IAction
	response chan ActionItemResponse
}

type ActionItemResponse struct {
	err error
}
