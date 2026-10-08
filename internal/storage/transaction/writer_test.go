package transaction

import (
	"testing"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransactionPatchSetter_OmittedFieldsAreNotWritten(t *testing.T) {
	categoryID := uuid.Must(uuid.NewV4())
	setter := transactionPatchSetter(&TransactionPatch{CategoryID: &categoryID})
	assert.Equal(t, []string{"category_id"}, setter.SetColumns())
	assert.Equal(t, categoryID, setter.CategoryID.MustGet())
	assert.Empty(t, transactionPatchSetter(&TransactionPatch{}).SetColumns())
}

func TestTransactionPatchSetter_ZeroAmountAndEmptyLabelsAreExplicitUpdates(t *testing.T) {
	amount := decimal.Zero
	name, merchant := "", ""
	date := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	setter := transactionPatchSetter(&TransactionPatch{
		Amount: &amount, TransactionName: &name, MerchantName: &merchant, TransactionDate: &date,
	})
	assert.ElementsMatch(t, []string{"amount", "transaction_name", "merchant_name", "transaction_date"}, setter.SetColumns())
	assert.True(t, amount.Equal(setter.Amount.MustGet()))
	assert.Equal(t, "", setter.TransactionName.MustGet())
	assert.Equal(t, "", setter.MerchantName.MustGet())
	assert.Equal(t, date, setter.TransactionDate.MustGet())
}
