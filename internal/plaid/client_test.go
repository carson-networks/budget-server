package plaid

import (
	"testing"

	plaidlib "github.com/plaid/plaid-go/v47/plaid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerchantNameFromPlaid_Set(t *testing.T) {
	txn := plaidlib.Transaction{}
	txn.SetMerchantName("Coffee Shop")

	got := merchantNameFromPlaid(txn)
	require.NotNil(t, got)
	assert.Equal(t, "Coffee Shop", *got)
}

func TestMerchantNameFromPlaid_ExplicitNull(t *testing.T) {
	txn := plaidlib.Transaction{}
	txn.SetMerchantNameNil()

	assert.Nil(t, merchantNameFromPlaid(txn))
}

func TestMerchantNameFromPlaid_Unset(t *testing.T) {
	txn := plaidlib.Transaction{}
	assert.Nil(t, merchantNameFromPlaid(txn))
}

func TestMapPlaidTransaction_IncludesMerchantName(t *testing.T) {
	txn := plaidlib.Transaction{
		TransactionId: "txn-1",
		AccountId:     "acc-1",
		Amount:        12.34,
		Name:          "STARBUCKS #123",
		Date:          "2025-03-01",
		Pending:       false,
	}
	txn.SetMerchantName("Starbucks")

	got := mapPlaidTransaction(txn)
	assert.Equal(t, "txn-1", got.PlaidTransactionID)
	assert.Equal(t, "acc-1", got.PlaidAccountID)
	assert.Equal(t, 12.34, got.Amount)
	assert.Equal(t, "STARBUCKS #123", got.Name)
	require.NotNil(t, got.MerchantName)
	assert.Equal(t, "Starbucks", *got.MerchantName)
	assert.Equal(t, "2025-03-01", got.Date)
	assert.False(t, got.Pending)
}

func TestMapPlaidTransaction_NullMerchantName(t *testing.T) {
	txn := plaidlib.Transaction{
		TransactionId: "txn-2",
		AccountId:     "acc-1",
		Amount:        100,
		Name:          "CHECK #4521",
		Date:          "2025-03-02",
		Pending:       false,
	}
	txn.SetMerchantNameNil()

	got := mapPlaidTransaction(txn)
	assert.Equal(t, "CHECK #4521", got.Name)
	assert.Nil(t, got.MerchantName)
}
