package providers

import (
	"testing"

	"github.com/gofrs/uuid/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
)

func TestBuildAddAction_MapsMerchantName(t *testing.T) {
	p := NewPlaidProvider(nil)
	accountID := uuid.Must(uuid.NewV4())
	merchant := "Uber"

	action, err := p.buildAddAction(plaidclient.SyncTransaction{
		PlaidTransactionID: "p-txn-1",
		PlaidAccountID:     "p-acc-1",
		Amount:             15.50,
		Name:               "UBER   TRIP",
		MerchantName:       &merchant,
		Date:               "2025-03-01",
	}, accountID)
	require.NoError(t, err)

	add, ok := action.(*actions.PlaidAddTransaction)
	require.True(t, ok)
	assert.Equal(t, "UBER   TRIP", add.Name)
	require.NotNil(t, add.MerchantName)
	assert.Equal(t, "Uber", *add.MerchantName)
	assert.Equal(t, accountID, add.AccountID)
}

func TestBuildAddAction_NilMerchantName(t *testing.T) {
	p := NewPlaidProvider(nil)
	accountID := uuid.Must(uuid.NewV4())

	action, err := p.buildAddAction(plaidclient.SyncTransaction{
		PlaidTransactionID: "p-txn-2",
		PlaidAccountID:     "p-acc-1",
		Amount:             50,
		Name:               "WIRE TRANSFER",
		MerchantName:       nil,
		Date:               "2025-03-02",
	}, accountID)
	require.NoError(t, err)

	add, ok := action.(*actions.PlaidAddTransaction)
	require.True(t, ok)
	assert.Nil(t, add.MerchantName)
}
