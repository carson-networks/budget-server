package plaid

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

// PlaidItem represents a single Plaid Item — one institution connection with one access token.
type PlaidItem struct {
	ID              uuid.UUID
	AccessToken     string
	PlaidItemID     string
	InstitutionID   string
	InstitutionName string
	Cursor          string
	CreatedAt       time.Time
}

// PlaidItemCreate is the input for creating a new PlaidItem.
type PlaidItemCreate struct {
	AccessToken     string
	PlaidItemID     string
	InstitutionID   string
	InstitutionName string
}

// AccountLink maps a Plaid account ID to an internal budget account.
type AccountLink struct {
	PlaidAccountID string
	AccountID      uuid.UUID
	PlaidItemID    uuid.UUID
}

// TransactionLink maps a Plaid transaction ID to an internal budget transaction.
// PlaidAccountID is denormalized for efficient removal lookups.
type TransactionLink struct {
	PlaidTransactionID string
	TransactionID      uuid.UUID
	PlaidAccountID     string
}
