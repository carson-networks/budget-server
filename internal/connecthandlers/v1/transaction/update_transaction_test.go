package v1Transaction

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
	connecthandlers "github.com/carson-networks/budget-server/internal/connecthandlers/v1"
	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
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
		{"category only", &transaction.UpdateTransactionRequest{Id: id.String(), CategoryId: proto.String(categoryID.String())}, &actions.UpdateTransaction{ID: id, CategoryID: &categoryID}},
		{"zero amount", &transaction.UpdateTransactionRequest{Id: id.String(), Amount: proto.String("0")}, &actions.UpdateTransaction{ID: id, Amount: &zero}},
		{"empty name", &transaction.UpdateTransactionRequest{Id: id.String(), TransactionName: proto.String("")}, &actions.UpdateTransaction{ID: id, TransactionName: proto.String("")}},
		{"empty merchant", &transaction.UpdateTransactionRequest{Id: id.String(), MerchantName: proto.String("")}, &actions.UpdateTransaction{ID: id, MerchantName: proto.String("")}},
		{"date only", &transaction.UpdateTransactionRequest{Id: id.String(), TransactionDate: timestamppb.New(date)}, &actions.UpdateTransaction{ID: id, TransactionDate: &date}},
		{"all fields", &transaction.UpdateTransactionRequest{
			Id: id.String(), CategoryId: proto.String(categoryID.String()), Amount: proto.String("0"),
			TransactionName: proto.String("Updated"), MerchantName: proto.String("Merchant"), TransactionDate: timestamppb.New(date),
		}, &actions.UpdateTransaction{
			ID: id, CategoryID: &categoryID, Amount: &zero, TransactionName: proto.String("Updated"),
			MerchantName: proto.String("Merchant"), TransactionDate: &date,
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			processor := operator.NewMockIProcessor(t)
			processor.EXPECT().Process(mock.Anything, test.action).Return(nil).Once()
			service := &Service{Deps: connecthandlers.Deps{Operator: processor}}
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
		{"missing id", &transaction.UpdateTransactionRequest{Amount: proto.String("1")}},
		{"invalid id", &transaction.UpdateTransactionRequest{Id: "invalid", Amount: proto.String("1")}},
		{"nil UUID", &transaction.UpdateTransactionRequest{Id: uuid.Nil.String(), Amount: proto.String("1")}},
		{"no edits", &transaction.UpdateTransactionRequest{Id: id}},
		{"invalid category", &transaction.UpdateTransactionRequest{Id: id, CategoryId: proto.String("invalid")}},
		{"empty category", &transaction.UpdateTransactionRequest{Id: id, CategoryId: proto.String("")}},
		{"nil category UUID", &transaction.UpdateTransactionRequest{Id: id, CategoryId: proto.String(uuid.Nil.String())}},
		{"invalid amount", &transaction.UpdateTransactionRequest{Id: id, Amount: proto.String("not-a-number")}},
		{"empty amount", &transaction.UpdateTransactionRequest{Id: id, Amount: proto.String("")}},
		{"excess precision", &transaction.UpdateTransactionRequest{Id: id, Amount: proto.String("1.00001")}},
		{"invalid timestamp", &transaction.UpdateTransactionRequest{Id: id, TransactionDate: &timestamppb.Timestamp{Nanos: -1}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			service := &Service{Deps: connecthandlers.Deps{Operator: operator.NewMockIProcessor(t)}}
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
		{actions.ErrTransactionNotFound, connect.CodeNotFound},
		{actions.ErrCategoryNotFoundForTransaction, connect.CodeNotFound},
		{actions.ErrAccountNotFound, connect.CodeNotFound},
		{actions.ErrCategoryDisabled, connect.CodeInvalidArgument},
		{actions.ErrCategoryIsParent, connect.CodeInvalidArgument},
		{errors.New("database failure"), connect.CodeInternal},
	} {
		t.Run(test.err.Error(), func(t *testing.T) {
			processor := operator.NewMockIProcessor(t)
			processor.EXPECT().Process(mock.Anything, mock.Anything).Return(test.err).Once()
			service := &Service{Deps: connecthandlers.Deps{Operator: processor}}
			_, err := service.UpdateTransaction(context.Background(), connect.NewRequest(&transaction.UpdateTransactionRequest{
				Id: uuid.Must(uuid.NewV4()).String(), TransactionName: proto.String("Updated"),
			}))
			require.Error(t, err)
			assert.Equal(t, test.code, connect.CodeOf(err))
		})
	}
}

func TestUpdateTransaction_RPCPreservesFieldPresence(t *testing.T) {
	id, categoryID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	processor := operator.NewMockIProcessor(t)
	processor.EXPECT().Process(mock.Anything, &actions.UpdateTransaction{ID: id, CategoryID: &categoryID}).Return(nil).Once()
	service := &Service{Deps: connecthandlers.Deps{Operator: processor}}
	_, handler := transaction.NewTransactionServiceHandler(service)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := transaction.NewTransactionServiceClient(server.Client(), server.URL)
	_, err := client.UpdateTransaction(context.Background(), connect.NewRequest(&transaction.UpdateTransactionRequest{
		Id: id.String(), CategoryId: proto.String(categoryID.String()),
	}))
	require.NoError(t, err)
}
