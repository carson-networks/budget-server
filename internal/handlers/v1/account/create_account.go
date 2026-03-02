package account

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage/account"
	"github.com/shopspring/decimal"
)

// CreateAccountBody is the request body for creating an account.
type CreateAccountBody struct {
	Name            string `json:"name" required:"true" doc:"Account name"`
	Type            int    `json:"type" doc:"Account type (0=Cash, 1=CreditCards, 2=Investments, 3=Loans, 4=Assets)"`
	SubType         string `json:"subType" doc:"Account sub-type"`
	StartingBalance string `json:"startingBalance" doc:"Starting balance as decimal string"`
}

// CreateAccountInput is the Huma input for creating an account.
type CreateAccountInput struct {
	Body CreateAccountBody
}

// CreateAccountOutput is the Huma output for creating an account.
type CreateAccountOutput struct {
	Status int `json:"status" doc:"HTTP status"`
}

// CreateAccountHandler handles POST /v1/accounts.
type CreateAccountHandler struct {
	Operator *operator.OperatorDelegator
}

// NewCreateAccountHandler creates a new CreateAccountHandler.
func NewCreateAccountHandler(op *operator.OperatorDelegator) *CreateAccountHandler {
	return &CreateAccountHandler{Operator: op}
}

// Register registers the create account endpoint with the Huma API.
func (h *CreateAccountHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-account",
		Method:      http.MethodPost,
		Path:        "/v1/accounts",
		Summary:     "Create account",
		Description: "Creates a new account.",
		Tags:        []string{"Accounts"},
	}, h.handle)
}

func (h *CreateAccountHandler) handle(ctx context.Context, input *CreateAccountInput) (*CreateAccountOutput, error) {
	startingBalance, err := decimal.NewFromString(input.Body.StartingBalance)
	if err != nil {
		startingBalance = decimal.Zero
	}

	action := &actions.CreateAccount{
		Name:            input.Body.Name,
		Type:            account.AccountType(input.Body.Type),
		SubType:         input.Body.SubType,
		StartingBalance: startingBalance,
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to create account", err)
	}

	return &CreateAccountOutput{Status: http.StatusCreated}, nil
}
