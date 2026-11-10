package plaid

import (
	"context"
	"errors"
	"fmt"
	"time"

	plaidlib "github.com/plaid/plaid-go/v50"
	plaidclient "github.com/plaid/plaid-go/v50/client"
	"github.com/plaid/plaid-go/v50/core"
	"github.com/plaid/plaid-go/v50/option"
)

type Client struct {
	api *plaidclient.Client
}

func NewClient(clientID, secret, env string) *Client {
	baseURL := plaidlib.Environments.Sandbox
	if env == "production" {
		baseURL = plaidlib.Environments.Production
	}

	api := plaidclient.NewClient(
		option.WithClientID(plaidlib.String(clientID)),
		option.WithSecret(secret),
		option.WithBaseURL(baseURL),
	)
	return &Client{api: api}
}

func (c *Client) CreateLinkToken(ctx context.Context) (string, time.Time, error) {
	resp, err := c.api.Link.CreateToken(ctx, &plaidlib.LinkTokenCreateRequest{
		ClientName:   "Budget",
		Language:     "en",
		CountryCodes: []plaidlib.CountryCode{plaidlib.CountryCodeUs},
		User:         &plaidlib.LinkTokenCreateRequestUser{ClientUserID: "budget-user"},
		Products:     []plaidlib.Products{plaidlib.ProductsTransactions},
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("plaid link token create: %w", plaidError(err))
	}
	return resp.LinkToken, resp.Expiration, nil
}

func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, plaidItemID string, err error) {
	resp, exchErr := c.api.Item.ExchangePublicToken(ctx, &plaidlib.ItemPublicTokenExchangeRequest{
		PublicToken: publicToken,
	})
	if exchErr != nil {
		return "", "", fmt.Errorf("plaid token exchange: %w", plaidError(exchErr))
	}
	return resp.AccessToken, resp.ItemID, nil
}

type SyncTransaction struct {
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             float64 // Plaid convention: positive = debit (money leaving account)
	Name               string
	MerchantName       *string // nil when Plaid omits or nulls merchant_name
	Date               string  // "YYYY-MM-DD"
	Pending            bool
}

// merchantNameFromPlaid returns a copy of Plaid's merchant_name when set to a non-null value.
func merchantNameFromPlaid(t plaidlib.Transaction) *string {
	if t.MerchantName == nil {
		return nil
	}
	s := *t.MerchantName
	return &s
}

func mapPlaidTransaction(t plaidlib.Transaction) SyncTransaction {
	return SyncTransaction{
		PlaidTransactionID: t.TransactionID,
		PlaidAccountID:     t.AccountID,
		Amount:             t.Amount,
		Name:               t.Name,
		MerchantName:       merchantNameFromPlaid(t),
		Date:               t.Date.Format(time.DateOnly),
		Pending:            t.Pending,
	}
}

type SyncResult struct {
	Added      []SyncTransaction
	Modified   []SyncTransaction
	Removed    []string // plaid_transaction_id strings to delete
	NextCursor string
	HasMore    bool
}

// Pass an empty string for cursor on the first call to fetch all history.
func (c *Client) SyncTransactions(ctx context.Context, accessToken, cursor string) (*SyncResult, error) {
	count := 500
	request := &plaidlib.TransactionsSyncRequest{
		AccessToken: accessToken,
		Count:       &count,
	}
	if cursor != "" {
		request.Cursor = &cursor
	}

	resp, err := c.api.Transactions.Sync(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("plaid transactions sync: %w", plaidError(err))
	}

	result := &SyncResult{
		NextCursor: resp.NextCursor,
		HasMore:    resp.HasMore,
	}

	for _, t := range resp.Added {
		result.Added = append(result.Added, mapPlaidTransaction(*t))
	}

	for _, t := range resp.Modified {
		result.Modified = append(result.Modified, mapPlaidTransaction(*t))
	}

	for _, r := range resp.Removed {
		result.Removed = append(result.Removed, r.TransactionID)
	}

	return result, nil
}

type AccountBalance struct {
	PlaidAccountID      string
	CurrentBalance      float64
	HasCurrentBalance   bool // true when Plaid returned a non-null current balance
	AvailableBalance    float64
	HasAvailableBalance bool
}

func (c *Client) GetAccountBalances(ctx context.Context, accessToken string) ([]AccountBalance, error) {
	resp, err := c.api.Accounts.Get(ctx, &plaidlib.AccountsGetRequest{AccessToken: accessToken})
	if err != nil {
		return nil, fmt.Errorf("plaid accounts get: %w", plaidError(err))
	}

	balances := make([]AccountBalance, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		b := AccountBalance{
			PlaidAccountID: a.AccountID,
		}
		if a.Balances != nil && a.Balances.Current != nil {
			b.CurrentBalance = *a.Balances.Current
			b.HasCurrentBalance = true
		}
		if a.Balances != nil && a.Balances.Available != nil {
			b.AvailableBalance = *a.Balances.Available
			b.HasAvailableBalance = true
		}
		balances = append(balances, b)
	}
	return balances, nil
}

func plaidError(err error) error {
	var plaidErr interface{ GetBody() *plaidlib.PlaidError }
	if !errors.As(err, &plaidErr) {
		return err
	}
	body := plaidErr.GetBody()
	if body == nil {
		return err
	}

	status := 0
	var apiErr *core.APIError
	if errors.As(err, &apiErr) {
		status = apiErr.StatusCode
	}
	return fmt.Errorf("%w (status %d): type=%s code=%s message=%s", err, status, body.ErrorType, body.ErrorCode, body.ErrorMessage)
}
