package account

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/service"
)

// CreateAccountInput is the Huma input for creating an account.
type CreateAccountInput struct {
	Body CreateAccountBody
}

// CreateAccountBody is the request body fields for creating an account.
type CreateAccountBody struct {
	Name            string `json:"name" minLength:"1" doc:"Account name"`
	Type            int    `json:"type" minimum:"0" maximum:"4" doc:"Account type: 0=Cash, 1=Credit Cards, 2=Investments, 3=Loans, 4=Assets"`
	SubType         string `json:"subType" doc:"Account sub-type"`
	Balance         string `json:"balance,omitempty" doc:"Initial current balance (e.g. '0' or '1234.56'), defaults to startingBalance"`
	StartingBalance string `json:"startingBalance,omitempty" doc:"Starting balance when account is created (e.g. '0' or '1234.56'), defaults to 0"`
}

// CreateAccountResponse is the response body for creating an account.
type CreateAccountResponse struct {
	ID string `json:"id" doc:"Created account UUID"`
}

// CreateAccountOutput is the response for creating an account.
type CreateAccountOutput struct {
	Status int
	Body   CreateAccountResponse
}

// accountCreator is the interface for creating accounts.
type accountCreator interface {
	CreateAccount(ctx context.Context, account service.Account) (uuid.UUID, error)
}

// CreateAccountHandler handles POST /v1/account.
type CreateAccountHandler struct {
	AccountService accountCreator
}

// NewCreateAccountHandler creates a new CreateAccountHandler.
func NewCreateAccountHandler(svc accountCreator) *CreateAccountHandler {
	return &CreateAccountHandler{AccountService: svc}
}

// Register registers the create account endpoint with the Huma API.
func (h *CreateAccountHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-account",
		Method:      http.MethodPost,
		Path:        "/v1/account",
		Summary:     "Create an account",
		Description: "Creates a new account with the given name, type, sub-type, and initial balance.",
		Tags:        []string{"Accounts"},
	}, h.handle)
}

func parseCreateAccountInput(input *CreateAccountInput) (service.Account, error) {
	startingBalanceStr := input.Body.StartingBalance
	if startingBalanceStr == "" {
		startingBalanceStr = "0"
	}
	startingBalance, err := decimal.NewFromString(startingBalanceStr)
	if err != nil {
		return service.Account{}, huma.NewError(http.StatusBadRequest, "invalid startingBalance", err)
	}

	balanceStr := input.Body.Balance
	if balanceStr == "" {
		balanceStr = startingBalanceStr
	}
	balance, err := decimal.NewFromString(balanceStr)
	if err != nil {
		return service.Account{}, huma.NewError(http.StatusBadRequest, "invalid balance", err)
	}

	if input.Body.Type < 0 || input.Body.Type > 4 {
		return service.Account{}, huma.NewError(http.StatusBadRequest, "type must be 0-4", nil)
	}

	return service.Account{
		Name:            input.Body.Name,
		Type:            service.AccountType(input.Body.Type),
		SubType:         input.Body.SubType,
		Balance:         balance,
		StartingBalance: startingBalance,
	}, nil
}

func (h *CreateAccountHandler) handle(ctx context.Context, input *CreateAccountInput) (*CreateAccountOutput, error) {
	logData := logging.GetLogData(ctx)

	account, err := parseCreateAccountInput(input)
	if err != nil {
		return nil, err
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("createAccountMs")
	}
	id, err := h.AccountService.CreateAccount(ctx, account)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to create account", err)
	}

	if logData != nil {
		logData.AddData("accountID", id.String())
	}

	return &CreateAccountOutput{
		Status: http.StatusCreated,
		Body:   CreateAccountResponse{ID: id.String()},
	}, nil
}
