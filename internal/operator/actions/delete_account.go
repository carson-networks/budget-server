package actions

import (
	"context"
	"database/sql"
	"errors"

	"github.com/carson-networks/budget-server/internal/storage"
	"github.com/gofrs/uuid/v5"
)

type DeleteAccount struct {
	ID uuid.UUID

	IAction
}

func (d *DeleteAccount) Perform(ctx context.Context, writer *storage.Writer) error {
	existing, err := writer.Account.FindByIDForUpdate(ctx, d.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrAccountNotFound
		}
		return err
	}
	if existing == nil {
		return ErrAccountNotFound
	}

	// Related rows cascade from accounts:
	// - transactions (fk_transactions_account_id ON DELETE CASCADE)
	// - plaid_transaction_links (cascade from transactions)
	// - plaid_account_links, syncs (ON DELETE CASCADE)
	return writer.Account.Delete(ctx, d.ID)
}
