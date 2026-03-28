package plaid

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

// SelectedAccount is one Plaid account the user has chosen to import.
// The frontend gets this data from the Plaid Link onSuccess metadata.
type SelectedAccount struct {
	PlaidAccountID string          `json:"plaidAccountID" doc:"Plaid's stable account ID string"`
	Name           string          `json:"name" doc:"Display name for the account"`
	Type           int             `json:"type" doc:"Account type (0=Cash, 1=CreditCards, 2=Investments, 3=Loans, 4=Assets)"`
	SubType        string          `json:"subType" doc:"Account sub-type (e.g. checking, savings)"`
	Balance        decimal.Decimal `json:"balance" doc:"Current balance, used as starting balance"`
}

// ExchangeTokenBody is the request body for POST /v1/plaid/exchange-token.
type ExchangeTokenBody struct {
	PublicToken     string            `json:"publicToken" required:"true" doc:"One-time public token from Plaid Link"`
	InstitutionID   string            `json:"institutionID" required:"true" doc:"Plaid institution ID from Link metadata"`
	InstitutionName string            `json:"institutionName" required:"true" doc:"Institution display name from Link metadata"`
	Accounts        []SelectedAccount `json:"accounts" required:"true" doc:"Accounts the user selected to import"`
}

// ExchangeTokenInput is the Huma input for exchanging a public token.
type ExchangeTokenInput struct {
	Body ExchangeTokenBody
}

// ExchangeTokenOutput is the Huma output for the token exchange.
type ExchangeTokenOutput struct {
	Body struct {
		Status int `json:"status"`
	}
}

// tokenExchanger is the subset of plaid.Client the exchange-token handler needs.
type tokenExchanger interface {
	ExchangePublicToken(ctx context.Context, publicToken string) (accessToken, plaidItemID string, err error)
}

// accountSyncer is the subset of the sync orchestrator the exchange-token handler needs.
type accountSyncer interface {
	Sync(ctx context.Context, accountIDs []uuid.UUID) error
}

// ExchangeTokenHandler handles POST /v1/plaid/exchange-token.
type ExchangeTokenHandler struct {
	Operator     operator.IProcessor
	PlaidClient  tokenExchanger
	Orchestrator accountSyncer
}

func NewExchangeTokenHandler(op operator.IProcessor, client *plaidclient.Client, orchestrator accountSyncer) *ExchangeTokenHandler {
	return &ExchangeTokenHandler{Operator: op, PlaidClient: client, Orchestrator: orchestrator}
}

func (h *ExchangeTokenHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID:   "exchange-plaid-token",
		Method:        http.MethodPost,
		Path:          "/v1/plaid/exchange-token",
		Summary:       "Exchange Plaid public token",
		Description:   "Exchanges a Plaid Link public token for a durable access token and creates budget accounts for the selected Plaid accounts.",
		Tags:          []string{"Plaid"},
		DefaultStatus: http.StatusCreated,
	}, h.handle)
}

func (h *ExchangeTokenHandler) handle(ctx context.Context, input *ExchangeTokenInput) (*ExchangeTokenOutput, error) {
	if len(input.Body.Accounts) == 0 {
		return nil, huma.NewError(http.StatusBadRequest, "at least one account must be selected")
	}

	accessToken, plaidItemID, err := h.PlaidClient.ExchangePublicToken(ctx, input.Body.PublicToken)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to exchange Plaid token", err)
	}

	plaidAccounts := make([]actions.PlaidAccountToLink, len(input.Body.Accounts))
	for i, a := range input.Body.Accounts {
		plaidAccounts[i] = actions.PlaidAccountToLink{
			PlaidAccountID: a.PlaidAccountID,
			Name:           a.Name,
			Type:           account.AccountType(a.Type),
			SubType:        a.SubType,
			Balance:        a.Balance,
		}
	}

	action := &actions.LinkPlaidItem{
		AccessToken:     accessToken,
		PlaidItemID:     plaidItemID,
		InstitutionID:   input.Body.InstitutionID,
		InstitutionName: input.Body.InstitutionName,
		Accounts:        plaidAccounts,
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to link Plaid item", err)
	}

	if err := h.Orchestrator.Sync(ctx, action.CreatedAccountIDs); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "accounts linked but initial sync failed", err)
	}

	out := &ExchangeTokenOutput{}
	out.Body.Status = http.StatusCreated
	return out, nil
}
