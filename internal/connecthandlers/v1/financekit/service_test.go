package v1FinanceKit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	financekit "github.com/carson-networks/budget-server/internal/connecthandlers/gen/financekit/v1"
)

func TestService_EveryRPCIsUnimplemented(t *testing.T) {
	connectServer := connect.NewServer()
	financekit.RegisterFinanceKitServiceHandler(connectServer, &Service{})
	mux := http.NewServeMux()
	connecthttp.Mount(mux, connectServer)
	server := httptest.NewServer(mux)
	defer server.Close()
	client := financekit.NewFinanceKitServiceClient(connect.NewClient(connecthttp.NewTransport(server.Client(), server.URL)))
	ctx := context.Background()

	for _, test := range []struct {
		name string
		call func() error
	}{
		{
			name: "GetStatus",
			call: func() error {
				_, err := client.GetStatus(ctx, &financekit.GetStatusRequest{})
				return err
			},
		},

		{
			name: "ListAccounts",
			call: func() error {
				_, err := client.ListAccounts(ctx, &financekit.ListAccountsRequest{})
				return err
			},
		},

		{
			name: "LinkAccount",
			call: func() error {
				_, err := client.LinkAccount(ctx, &financekit.LinkAccountRequest{})
				return err
			},
		},

		{
			name: "UnlinkAccount",
			call: func() error {
				_, err := client.UnlinkAccount(ctx, &financekit.UnlinkAccountRequest{})
				return err
			},
		},

		{
			name: "PushChanges",
			call: func() error {
				_, err := client.PushChanges(ctx, &financekit.PushChangesRequest{})
				return err
			},
		},

		{
			name: "RegisterPushToken",
			call: func() error {
				_, err := client.RegisterPushToken(ctx, &financekit.RegisterPushTokenRequest{})
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.call()

			require.Error(t, err)
			assert.Equal(t, connect.CodeUnimplemented, connect.CodeOf(err))
		})
	}
}
