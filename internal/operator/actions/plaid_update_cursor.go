package actions

import (
	"context"

	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/storage"
)

type PlaidUpdateCursor struct {
	ItemID     uuid.UUID
	NextCursor string
}

func (a *PlaidUpdateCursor) Perform(ctx context.Context, writer *storage.Writer) error {
	return writer.Plaid.UpdateCursor(ctx, a.ItemID, a.NextCursor)
}
