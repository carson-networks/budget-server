package actions

import (
	"context"

	"github.com/carson-networks/budget-server/internal/storage"
)

type IAction interface {
	Perform(ctx context.Context, writer *storage.Writer) error
}
