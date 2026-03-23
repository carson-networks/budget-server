package plaid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	plaidlib "github.com/plaid/plaid-go/v41/plaid"
)

// Client wraps the Plaid SDK and exposes only the operations this server needs.
type Client struct {
	api *plaidlib.PlaidApiService
}

// NewClient creates a new Plaid API client for the given environment.
// env should be "sandbox", "development", or "production".
func NewClient(clientID, secret, env string) *Client {
	cfg := plaidlib.NewConfiguration()
	cfg.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	cfg.AddDefaultHeader("PLAID-SECRET", secret)

	switch env {
	case "production":
		cfg.UseEnvironment(plaidlib.Production)
	default:
		cfg.UseEnvironment(plaidlib.Sandbox)
	}

	apiClient := plaidlib.NewAPIClient(cfg)
	return &Client{api: apiClient.PlaidApi}
}

// CreateLinkToken creates a Plaid Link token for the frontend Plaid Link widget.
func (c *Client) CreateLinkToken(ctx context.Context) (string, time.Time, error) {
	request := plaidlib.NewLinkTokenCreateRequest(
		"Budget",
		"en",
		[]plaidlib.CountryCode{plaidlib.COUNTRYCODE_US},
	)
	user := plaidlib.NewLinkTokenCreateRequestUser("budget-user")
	request.SetUser(*user)
	request.SetProducts([]plaidlib.Products{plaidlib.PRODUCTS_TRANSACTIONS})

	resp, httpResp, err := c.api.LinkTokenCreate(ctx).LinkTokenCreateRequest(*request).Execute()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("plaid link token create: %w", plaidError(err, httpResp))
	}
	return resp.LinkToken, resp.Expiration, nil
}

// ExchangePublicToken exchanges the one-time public_token from the frontend for a durable
// access_token and plaid_item_id.
func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, plaidItemID string, err error) {
	request := plaidlib.NewItemPublicTokenExchangeRequest(publicToken)
	resp, httpResp, exchErr := c.api.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*request).Execute()
	if exchErr != nil {
		return "", "", fmt.Errorf("plaid token exchange: %w", plaidError(exchErr, httpResp))
	}
	return resp.AccessToken, resp.ItemId, nil
}

// SyncTransaction is a single transaction returned by Plaid's /transactions/sync.
type SyncTransaction struct {
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             float64 // Plaid convention: positive = debit (money leaving account)
	Name               string
	Date               string // "YYYY-MM-DD"
	Pending            bool
}

// SyncResult is the result of one page of /transactions/sync.
type SyncResult struct {
	Added      []SyncTransaction
	Modified   []SyncTransaction
	Removed    []string // plaid_transaction_id strings to delete
	NextCursor string
	HasMore    bool
}

// SyncTransactions calls /transactions/sync with the given cursor.
// Pass an empty string for cursor on the first call to fetch all history.
func (c *Client) SyncTransactions(ctx context.Context, accessToken, cursor string) (*SyncResult, error) {
	request := plaidlib.NewTransactionsSyncRequest(accessToken)
	if cursor != "" {
		request.SetCursor(cursor)
	}
	request.SetCount(500)

	resp, httpResp, err := c.api.TransactionsSync(ctx).TransactionsSyncRequest(*request).Execute()
	if err != nil {
		return nil, fmt.Errorf("plaid transactions sync: %w", plaidError(err, httpResp))
	}

	result := &SyncResult{
		NextCursor: resp.NextCursor,
		HasMore:    resp.HasMore,
	}

	for _, t := range resp.Added {
		result.Added = append(result.Added, SyncTransaction{
			PlaidTransactionID: t.TransactionId,
			PlaidAccountID:     t.AccountId,
			Amount:             t.Amount,
			Name:               t.Name,
			Date:               t.Date,
			Pending:            t.Pending,
		})
	}

	for _, t := range resp.Modified {
		result.Modified = append(result.Modified, SyncTransaction{
			PlaidTransactionID: t.TransactionId,
			PlaidAccountID:     t.AccountId,
			Amount:             t.Amount,
			Name:               t.Name,
			Date:               t.Date,
			Pending:            t.Pending,
		})
	}

	for _, r := range resp.Removed {
		result.Removed = append(result.Removed, r.TransactionId)
	}

	return result, nil
}

// plaidError extracts a meaningful error from a Plaid SDK error response.
func plaidError(err error, httpResp *http.Response) error {
	var plaidErr plaidlib.GenericOpenAPIError
	ok := errors.As(err, &plaidErr)
	if !ok {
		return err
	}
	if body := plaidErr.Body(); len(body) > 0 {
		return fmt.Errorf("%s (status %d): %s", plaidErr.Error(), httpResp.StatusCode, string(body))
	}
	return err
}
