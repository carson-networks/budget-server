package v1Transaction

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestUpdateTransaction_OptionalFields(t *testing.T) {
	id, categoryID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	zero := decimal.RequireFromString("0")
	date := time.Unix(0, 0).UTC()

	for _, test := range []struct {
		name    string
		request *transaction.UpdateTransactionRequest
		action  *actions.UpdateTransaction
	}{
		{
			name: "category only",
			request: &transaction.UpdateTransactionRequest{
				Id:         id.String(),
				CategoryId: new(categoryID.String()),
			},
			action: &actions.UpdateTransaction{
				ID:         id,
				CategoryID: new(categoryID),
			},
		},

		{
			name: "zero amount",
			request: &transaction.UpdateTransactionRequest{
				Id:     id.String(),
				Amount: new("0"),
			},
			action: &actions.UpdateTransaction{
				ID:     id,
				Amount: new(zero),
			},
		},

		{
			name: "empty name",
			request: &transaction.UpdateTransactionRequest{
				Id:              id.String(),
				TransactionName: new(""),
			},
			action: &actions.UpdateTransaction{
				ID:              id,
				TransactionName: new(""),
			},
		},

		{
			name: "empty merchant",
			request: &transaction.UpdateTransactionRequest{
				Id:           id.String(),
				MerchantName: new(""),
			},
			action: &actions.UpdateTransaction{
				ID:           id,
				MerchantName: new(""),
			},
		},

		{
			name: "date only",
			request: &transaction.UpdateTransactionRequest{
				Id:              id.String(),
				TransactionDate: timestamppb.New(date),
			},
			action: &actions.UpdateTransaction{
				ID:              id,
				TransactionDate: new(date),
			},
		},

		{
			name: "all fields",
			request: &transaction.UpdateTransactionRequest{
				Id:              id.String(),
				CategoryId:      new(categoryID.String()),
				Amount:          new("0"),
				TransactionName: new("Updated"),
				MerchantName:    new("Merchant"),
				TransactionDate: timestamppb.New(date),
			},
			action: &actions.UpdateTransaction{
				ID:              id,
				CategoryID:      new(categoryID),
				Amount:          new(zero),
				TransactionName: new("Updated"),
				MerchantName:    new("Merchant"),
				TransactionDate: new(date),
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			processor := operator.NewMockIProcessor(t)
			processor.EXPECT().Process(mock.Anything, test.action).Return(nil).Once()
			service := &Service{Operator: processor}

			response, err := service.UpdateTransaction(context.Background(), connect.NewRequest(test.request))

			require.NoError(t, err)
			require.NotNil(t, response)
		})
	}
}

func TestUpdateTransaction_RejectsInvalidInputBeforeProcessing(t *testing.T) {
	id := uuid.Must(uuid.NewV4()).String()

	for _, test := range []struct {
		name    string
		request *transaction.UpdateTransactionRequest
	}{
		{
			name: "missing id",
			request: &transaction.UpdateTransactionRequest{
				Amount: new("1"),
			},
		},

		{
			name: "invalid id",
			request: &transaction.UpdateTransactionRequest{
				Id:     "invalid",
				Amount: new("1"),
			},
		},

		{
			name: "nil UUID",
			request: &transaction.UpdateTransactionRequest{
				Id:     uuid.Nil.String(),
				Amount: new("1"),
			},
		},

		{
			name: "no edits",
			request: &transaction.UpdateTransactionRequest{
				Id: id,
			},
		},

		{
			name: "invalid category",
			request: &transaction.UpdateTransactionRequest{
				Id:         id,
				CategoryId: new("invalid"),
			},
		},

		{
			name: "empty category",
			request: &transaction.UpdateTransactionRequest{
				Id:         id,
				CategoryId: new(""),
			},
		},

		{
			name: "nil category UUID",
			request: &transaction.UpdateTransactionRequest{
				Id:         id,
				CategoryId: new(uuid.Nil.String()),
			},
		},

		{
			name: "invalid amount",
			request: &transaction.UpdateTransactionRequest{
				Id:     id,
				Amount: new("not-a-number"),
			},
		},

		{
			name: "empty amount",
			request: &transaction.UpdateTransactionRequest{
				Id:     id,
				Amount: new(""),
			},
		},

		{
			name: "excess precision",
			request: &transaction.UpdateTransactionRequest{
				Id:     id,
				Amount: new("1.00001"),
			},
		},

		{
			name: "invalid timestamp",
			request: &transaction.UpdateTransactionRequest{
				Id:              id,
				TransactionDate: &timestamppb.Timestamp{Nanos: -1},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &Service{Operator: operator.NewMockIProcessor(t)}

			_, err := service.UpdateTransaction(context.Background(), connect.NewRequest(test.request))

			require.Error(t, err)
			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		})
	}
}

func TestUpdateTransaction_MapsActionErrors(t *testing.T) {
	for _, test := range []struct {
		err  error
		code connect.Code
	}{
		{
			err:  actions.ErrTransactionNotFound,
			code: connect.CodeNotFound,
		},

		{
			err:  actions.ErrCategoryNotFoundForTransaction,
			code: connect.CodeNotFound,
		},

		{
			err:  actions.ErrAccountNotFound,
			code: connect.CodeNotFound,
		},

		{
			err:  actions.ErrCategoryDisabled,
			code: connect.CodeInvalidArgument,
		},

		{
			err:  actions.ErrCategoryIsParent,
			code: connect.CodeInvalidArgument,
		},

		{
			err:  errors.New("database failure"),
			code: connect.CodeInternal,
		},
	} {
		t.Run(test.err.Error(), func(t *testing.T) {
			processor := operator.NewMockIProcessor(t)
			processor.EXPECT().Process(mock.Anything, mock.Anything).Return(test.err).Once()
			service := &Service{Operator: processor}

			_, err := service.UpdateTransaction(context.Background(), connect.NewRequest(&transaction.UpdateTransactionRequest{
				Id:              uuid.Must(uuid.NewV4()).String(),
				TransactionName: new("Updated"),
			}))

			require.Error(t, err)
			assert.Equal(t, test.code, connect.CodeOf(err))
		})
	}
}

func TestUpdateTransaction_RPCPreservesFieldPresence(t *testing.T) {
	id, categoryID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	processor := operator.NewMockIProcessor(t)
	processor.EXPECT().Process(mock.Anything, &actions.UpdateTransaction{
		ID:         id,
		CategoryID: new(categoryID),
	}).Return(nil).Once()
	service := &Service{Operator: processor}

	_, handler := transaction.NewTransactionServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := transaction.NewTransactionServiceClient(server.Client(), server.URL)

	_, err := client.UpdateTransaction(context.Background(), connect.NewRequest(&transaction.UpdateTransactionRequest{
		Id:         id.String(),
		CategoryId: new(categoryID.String()),
	}))

	require.NoError(t, err)
}
