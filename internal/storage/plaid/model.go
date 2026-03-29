package plaid

import (
	"time"

	"github.com/gofrs/uuid/v5"
)

type PlaidItem struct {
	ID              uuid.UUID
	AccessToken     string
	PlaidItemID     string
	InstitutionID   string
	InstitutionName string
	Cursor          string
	CreatedAt       time.Time
}

type PlaidItemCreate struct {
	AccessToken     string
	PlaidItemID     string
	InstitutionID   string
	InstitutionName string
}

type AccountLink struct {
	PlaidAccountID string
	AccountID      uuid.UUID
	PlaidItemID    uuid.UUID
}

// PlaidAccountID is denormalized for efficient removal lookups.
type TransactionLink struct {
	PlaidTransactionID string
	TransactionID      uuid.UUID
	PlaidAccountID     string
}
