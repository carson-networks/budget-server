package v1Transaction

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	storagetransaction "github.com/carson-networks/budget-server/internal/storage/transaction"
)

func TestListTransactionsResponse_IncludesTotalCountAndCursor(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	created := time.Date(2025, 4, 1, 10, 0, 0, 0, time.UTC)
	maxCreation := time.Date(2025, 4, 2, 0, 0, 0, 0, time.UTC)

	result := &storagetransaction.TransactionListResult{
		Transactions: []*storagetransaction.Transaction{
			{
				ID:              id,
				AccountID:       accountID,
				Amount:          decimal.NewFromFloat(-12.34),
				TransactionName: "Coffee",
				TransactionDate: created,
				CreatedAt:       created,
			},
		},
		NextCursor: &storagetransaction.TransactionCursor{
			Position:        20,
			Limit:           20,
			MaxCreationTime: maxCreation,
		},
		TotalCount: 47,
	}

	out := listTransactionsResponse(result, result.Transactions)
	assert.Equal(t, int32(47), out.TotalCount)
	require.Len(t, out.Transactions, 1)
	assert.Equal(t, id.String(), out.Transactions[0].Id)
	require.NotNil(t, out.NextCursor)
	assert.Equal(t, int32(20), out.NextCursor.Position)
	assert.Equal(t, int32(20), out.NextCursor.Limit)
	assert.True(t, out.NextCursor.MaxCreationTime.AsTime().Equal(maxCreation))
}

func TestListTransactionsResponse_EmptyPageZeroTotal(t *testing.T) {
	result := &storagetransaction.TransactionListResult{
		Transactions: nil,
		NextCursor:   nil,
		TotalCount:   0,
	}
	out := listTransactionsResponse(result, []*storagetransaction.Transaction{})
	assert.Equal(t, int32(0), out.TotalCount)
	assert.Empty(t, out.Transactions)
	assert.Nil(t, out.NextCursor)
}

func TestListTransactionsResponse_LastPageKeepsTotalWithoutCursor(t *testing.T) {
	result := &storagetransaction.TransactionListResult{
		Transactions: []*storagetransaction.Transaction{
			{
				ID:              uuid.Must(uuid.NewV4()),
				AccountID:       uuid.Must(uuid.NewV4()),
				Amount:          decimal.NewFromInt(1),
				TransactionName: "Last",
				TransactionDate: time.Now().UTC(),
				CreatedAt:       time.Now().UTC(),
			},
		},
		NextCursor: nil,
		TotalCount: 21,
	}
	out := listTransactionsResponse(result, result.Transactions)
	assert.Equal(t, int32(21), out.TotalCount)
	require.Len(t, out.Transactions, 1)
	assert.Nil(t, out.NextCursor)
}

func TestListTransactionsFilter_AccountAndPagination(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	frozen := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	filter, err := listTransactionsFilter(&transaction.ListTransactionsRequest{
		AccountId: proto.String(accountID.String()),
		Cursor: &transaction.ListTransactionsCursor{
			Position: 50, Limit: 25, MaxCreationTime: timestamppb.New(frozen),
		},
	})
	require.NoError(t, err)
	require.NotNil(t, filter.AccountID)
	assert.Equal(t, accountID, *filter.AccountID)
	assert.Equal(t, 50, filter.Offset)
	assert.Equal(t, 25, filter.Limit)
	require.NotNil(t, filter.MaxCreationTime)
	assert.Equal(t, frozen, *filter.MaxCreationTime)
}

func TestListTransactionsFilter_UnfilteredDefaults(t *testing.T) {
	filter, err := listTransactionsFilter(&transaction.ListTransactionsRequest{})
	require.NoError(t, err)
	assert.Nil(t, filter.AccountID)
	assert.Equal(t, 0, filter.Offset)
	assert.Equal(t, 20, filter.Limit)
	assert.Nil(t, filter.MaxCreationTime)
}

func TestListTransactions_InvalidFilterRejectedBeforeStorage(t *testing.T) {
	for _, test := range []struct {
		name    string
		request *transaction.ListTransactionsRequest
	}{
		{"malformed account", &transaction.ListTransactionsRequest{AccountId: proto.String("invalid")}},
		{"empty account", &transaction.ListTransactionsRequest{AccountId: proto.String("")}},
		{"negative offset", &transaction.ListTransactionsRequest{Cursor: &transaction.ListTransactionsCursor{Position: -1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			// No storage dependency is needed: validation must run before accessing it.
			response, err := (&Service{}).ListTransactions(context.Background(), connect.NewRequest(test.request))
			assert.Nil(t, response)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}
