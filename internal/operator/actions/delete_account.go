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

	// transactions.account_id has no FK — delete explicitly so rows are not left dangling.
	// plaid_transaction_links cascade from transactions ON DELETE CASCADE.
	if err := writer.Transaction.DeleteByAccountID(ctx, d.ID); err != nil {
		return err
	}

	// plaid_account_links and syncs also CASCADE from accounts; delete explicitly
	// so related cleanup is clear and covered by action tests.
	if err := writer.Plaid.DeleteAccountLinksByAccountID(ctx, d.ID); err != nil {
		return err
	}
	if err := writer.Sync.Delete(ctx, d.ID); err != nil {
		return err
	}

	return writer.Account.Delete(ctx, d.ID)
}
