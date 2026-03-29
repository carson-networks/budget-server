package plaid

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	plaidlib "github.com/plaid/plaid-go/v41/plaid"
)

type Client struct {
	api *plaidlib.PlaidApiService
}

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

func (c *Client) ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, plaidItemID string, err error) {
	request := plaidlib.NewItemPublicTokenExchangeRequest(publicToken)
	resp, httpResp, exchErr := c.api.ItemPublicTokenExchange(ctx).ItemPublicTokenExchangeRequest(*request).Execute()
	if exchErr != nil {
		return "", "", fmt.Errorf("plaid token exchange: %w", plaidError(exchErr, httpResp))
	}
	return resp.AccessToken, resp.ItemId, nil
}

type SyncTransaction struct {
	PlaidTransactionID string
	PlaidAccountID     string
	Amount             float64 // Plaid convention: positive = debit (money leaving account)
	Name               string
	Date               string // "YYYY-MM-DD"
	Pending            bool
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

type AccountBalance struct {
	PlaidAccountID      string
	CurrentBalance      float64
	HasCurrentBalance   bool // true when Plaid returned a non-null current balance
	AvailableBalance    float64
	HasAvailableBalance bool
}

func (c *Client) GetAccountBalances(ctx context.Context, accessToken string) ([]AccountBalance, error) {
	request := plaidlib.NewAccountsGetRequest(accessToken)
	resp, httpResp, err := c.api.AccountsGet(ctx).AccountsGetRequest(*request).Execute()
	if err != nil {
		return nil, fmt.Errorf("plaid accounts get: %w", plaidError(err, httpResp))
	}

	balances := make([]AccountBalance, 0, len(resp.Accounts))
	for _, a := range resp.Accounts {
		b := AccountBalance{
			PlaidAccountID: a.AccountId,
		}
		if current, ok := a.Balances.GetCurrentOk(); ok && current != nil {
			b.CurrentBalance = *current
			b.HasCurrentBalance = true
		}
		if available, ok := a.Balances.GetAvailableOk(); ok && available != nil {
			b.AvailableBalance = *available
			b.HasAvailableBalance = true
		}
		balances = append(balances, b)
	}
	return balances, nil
}

func plaidError(err error, httpResp *http.Response) error {
	var plaidErr plaidlib.GenericOpenAPIError
	ok := errors.As(err, &plaidErr)
	if !ok {
		return err
	}
	if body := plaidErr.Body(); len(body) > 0 {
		status := 0
		if httpResp != nil {
			status = httpResp.StatusCode
		}
		return fmt.Errorf("%s (status %d): %s", plaidErr.Error(), status, string(body))
	}
	return err
}
