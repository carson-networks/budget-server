package plaid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	plaidlib "github.com/plaid/plaid-go/v50"
	plaidclient "github.com/plaid/plaid-go/v50/client"
	"github.com/plaid/plaid-go/v50/core"
	"github.com/plaid/plaid-go/v50/option"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMerchantNameFromPlaid_Set(t *testing.T) {
	name := "Coffee Shop"
	txn := plaidlib.Transaction{MerchantName: &name}

	got := merchantNameFromPlaid(txn)
	require.NotNil(t, got)
	assert.Equal(t, "Coffee Shop", *got)
	assert.NotSame(t, &name, got)
}

func TestMerchantNameFromPlaid_Nil(t *testing.T) {
	assert.Nil(t, merchantNameFromPlaid(plaidlib.Transaction{}))
}

func TestMapPlaidTransaction_IncludesMerchantName(t *testing.T) {
	merchant := "Starbucks"
	txn := plaidlib.Transaction{
		TransactionID: "txn-1",
		AccountID:     "acc-1",
		Amount:        12.34,
		Name:          "STARBUCKS #123",
		MerchantName:  &merchant,
		Date:          time.Date(2025, time.March, 1, 0, 0, 0, 0, time.UTC),
		Pending:       false,
	}

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
		TransactionID: "txn-2",
		AccountID:     "acc-1",
		Amount:        100,
		Name:          "CHECK #4521",
		Date:          time.Date(2025, time.March, 2, 0, 0, 0, 0, time.UTC),
		Pending:       false,
	}

	got := mapPlaidTransaction(txn)
	assert.Equal(t, "CHECK #4521", got.Name)
	assert.Nil(t, got.MerchantName)
	assert.Equal(t, "2025-03-02", got.Date)
}

func TestPlaidError_TypedErrorIncludesBodyAndStatus(t *testing.T) {
	err := &plaidlib.BadRequestError{
		APIError: core.NewAPIError(400, nil, errors.New("bad request")),
		Body: &plaidlib.PlaidError{
			ErrorCode:    "ITEM_LOGIN_REQUIRED",
			ErrorMessage: "the login details of this item have changed",
		},
	}

	got := plaidError(err)
	require.Error(t, got)
	assert.ErrorIs(t, got, err)
	assert.Contains(t, got.Error(), "status 400")
	assert.Contains(t, got.Error(), "ITEM_LOGIN_REQUIRED")
	assert.Contains(t, got.Error(), "the login details of this item have changed")
}

func TestPlaidError_UntypedErrorPassesThrough(t *testing.T) {
	orig := errors.New("connection refused")
	assert.Same(t, orig, plaidError(orig))
}

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Client{api: plaidclient.NewClient(
		option.WithClientID(plaidlib.String("test-id")),
		option.WithSecret("test-secret"),
		option.WithCustomEndpoint(srv.URL, srv.Client()),
	)}
}

func TestSyncTransactions_RequestAndResponse(t *testing.T) {
	var gotPath string
	var gotBody map[string]any
	var gotHeaders http.Header
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotHeaders = r.Header.Clone()
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{
			"added": [{"transaction_id":"t1","account_id":"a1","amount":5.5,"name":"N","merchant_name":"M","date":"2025-03-01","pending":true}],
			"modified": [{"transaction_id":"t2","account_id":"a1","amount":1,"name":"N2","merchant_name":null,"date":"2025-03-02","pending":false}],
			"removed": [{"transaction_id":"t3","account_id":"a1"}],
			"next_cursor": "c2",
			"has_more": true
		}`))
	})

	got, err := c.SyncTransactions(context.Background(), "access-token", "")
	require.NoError(t, err)

	assert.Equal(t, "/transactions/sync", gotPath)
	assert.Equal(t, "test-id", gotHeaders.Get("PLAID-CLIENT-ID"))
	assert.Equal(t, "test-secret", gotHeaders.Get("PLAID-SECRET"))
	assert.Equal(t, "access-token", gotBody["access_token"])
	assert.EqualValues(t, 500, gotBody["count"])
	assert.NotContains(t, gotBody, "cursor")

	require.Len(t, got.Added, 1)
	assert.Equal(t, "t1", got.Added[0].PlaidTransactionID)
	assert.Equal(t, "2025-03-01", got.Added[0].Date)
	assert.True(t, got.Added[0].Pending)
	require.NotNil(t, got.Added[0].MerchantName)
	assert.Equal(t, "M", *got.Added[0].MerchantName)
	require.Len(t, got.Modified, 1)
	assert.Nil(t, got.Modified[0].MerchantName)
	assert.Equal(t, []string{"t3"}, got.Removed)
	assert.Equal(t, "c2", got.NextCursor)
	assert.True(t, got.HasMore)
}

func TestSyncTransactions_SendsCursor(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{"added":[],"modified":[],"removed":[],"next_cursor":"c3","has_more":false}`))
	})

	_, err := c.SyncTransactions(context.Background(), "access-token", "c2")
	require.NoError(t, err)
	assert.Equal(t, "c2", gotBody["cursor"])
}

func TestGetAccountBalances(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/accounts/get", r.URL.Path)
		_, _ = w.Write([]byte(`{"accounts":[
			{"account_id":"a1","name":"Checking","type":"depository","balances":{"current":10.5,"available":null}},
			{"account_id":"a2","name":"Card","type":"credit","balances":{"current":null,"available":3}}
		],"item":{"item_id":"i1"},"request_id":"r"}`))
	})

	got, err := c.GetAccountBalances(context.Background(), "access-token")
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, AccountBalance{PlaidAccountID: "a1", CurrentBalance: 10.5, HasCurrentBalance: true}, got[0])
	assert.Equal(t, AccountBalance{PlaidAccountID: "a2", AvailableBalance: 3, HasAvailableBalance: true}, got[1])
}

func TestCreateLinkToken(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/link/token/create", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{"link_token":"link-1","expiration":"2025-03-01T12:00:00Z","request_id":"r"}`))
	})

	token, exp, err := c.CreateLinkToken(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "link-1", token)
	assert.True(t, exp.Equal(time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC)))
	assert.Equal(t, "Budget", gotBody["client_name"])
	assert.Equal(t, "en", gotBody["language"])
	assert.Equal(t, []any{"US"}, gotBody["country_codes"])
	assert.Equal(t, []any{"transactions"}, gotBody["products"])
	assert.Equal(t, map[string]any{"client_user_id": "budget-user"}, gotBody["user"])
}

func TestExchangePublicToken(t *testing.T) {
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/item/public_token/exchange", r.URL.Path)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		_, _ = w.Write([]byte(`{"access_token":"acc-1","item_id":"item-1","request_id":"r"}`))
	})

	access, item, err := c.ExchangePublicToken(context.Background(), "public-1")
	require.NoError(t, err)
	assert.Equal(t, "acc-1", access)
	assert.Equal(t, "item-1", item)
	assert.Equal(t, "public-1", gotBody["public_token"])
}

func TestSyncTransactions_PlaidAPIError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error_type":"ITEM_ERROR","error_code":"ITEM_LOGIN_REQUIRED","error_message":"login changed","status":400}`))
	})

	_, err := c.SyncTransactions(context.Background(), "access-token", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
	assert.Contains(t, err.Error(), "ITEM_LOGIN_REQUIRED")

	var badReq *plaidlib.BadRequestError
	assert.True(t, errors.As(err, &badReq))
}
