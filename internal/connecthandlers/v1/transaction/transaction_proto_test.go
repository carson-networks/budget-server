package v1Transaction

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

func TestTransactionToProto_IncludesMerchantName(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	categoryID := uuid.Must(uuid.NewV4())
	merchant := "Starbucks"
	date := time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)
	created := time.Date(2025, 3, 1, 13, 0, 0, 0, time.UTC)

	got := transactionToProto(&storagetransaction.Transaction{
		ID:              id,
		AccountID:       accountID,
		CategoryID:      &categoryID,
		Amount:          decimal.NewFromFloat(-4.50),
		TransactionName: "STARBUCKS STORE 123",
		MerchantName:    &merchant,
		TransactionDate: date,
		CreatedAt:       created,
	})

	require.NotNil(t, got.MerchantName)
	assert.Equal(t, merchant, *got.MerchantName)
	assert.Equal(t, "STARBUCKS STORE 123", got.TransactionName)
	assert.Equal(t, id.String(), got.Id)
	assert.Equal(t, accountID.String(), got.AccountId)
	require.NotNil(t, got.CategoryId)
	assert.Equal(t, categoryID.String(), *got.CategoryId)
}

func TestTransactionToProto_NilMerchantName(t *testing.T) {
	got := transactionToProto(&storagetransaction.Transaction{
		ID:              uuid.Must(uuid.NewV4()),
		AccountID:       uuid.Must(uuid.NewV4()),
		Amount:          decimal.NewFromFloat(10),
		TransactionName: "ACH TRANSFER",
		MerchantName:    nil,
		TransactionDate: time.Now().UTC(),
		CreatedAt:       time.Now().UTC(),
	})

	assert.Nil(t, got.MerchantName)
	assert.Equal(t, "ACH TRANSFER", got.TransactionName)
}
