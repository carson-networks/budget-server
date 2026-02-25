package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/service"
)

// CreateTransactionInput is the Huma input for creating a transaction.
type CreateTransactionInput struct {
	Body CreateTransactionBody
}

// CreateTransactionBody is the request body fields for creating a transaction.
type CreateTransactionBody struct {
	AccountID       string `json:"accountID" format:"uuid" doc:"Account UUID"`
	CategoryID      string `json:"categoryID" format:"uuid" doc:"Category UUID"`
	Amount          string `json:"amount" doc:"Decimal amount (e.g. '12.50')"`
	TransactionName string `json:"transactionName" minLength:"1" doc:"Name of the transaction"`
	TransactionDate string `json:"transactionDate,omitempty" format:"date-time" doc:"RFC3339 date, defaults to now"`
}

// CreateTransactionResponse is the response body for creating a transaction.
type CreateTransactionResponse struct {
	ID string `json:"id" doc:"Created transaction UUID"`
}

// CreateTransactionOutput is the response for creating a transaction.
type CreateTransactionOutput struct {
	Status int
	Body   CreateTransactionResponse
}

// transactionCreator is the interface for creating transactions.
// It allows injecting a mock in tests.
type transactionCreator interface {
	CreateTransaction(ctx context.Context, transaction service.Transaction) (uuid.UUID, error)
}

// CreateTransactionHandler handles POST /v1/transaction.
type CreateTransactionHandler struct {
	TransactionService transactionCreator
}

// NewCreateTransactionHandler creates a new CreateTransactionHandler.
func NewCreateTransactionHandler(svc transactionCreator) *CreateTransactionHandler {
	return &CreateTransactionHandler{TransactionService: svc}
}

// Register registers the create transaction endpoint with the Huma API.
func (h *CreateTransactionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-transaction",
		Method:      http.MethodPost,
		Path:        "/v1/transaction",
		Summary:     "Create a transaction",
		Description: "Creates a new transaction with the given account, category, amount, name, and optional date.",
		Tags:        []string{"Transactions"},
	}, h.handle)
}

// parseCreateTransactionInput parses and validates the API input.
// Returns individual fields or a huma error on validation failure.
func parseCreateTransactionInput(input *CreateTransactionInput) (accountID uuid.UUID, categoryID uuid.UUID, amount decimal.Decimal, transactionName string, transactionDate time.Time, err error) {
	accountID, err = uuid.FromString(input.Body.AccountID)
	if err != nil {
		return uuid.Nil, uuid.Nil, decimal.Decimal{}, "", time.Time{}, huma.NewError(http.StatusBadRequest, "invalid accountID", err)
	}

	categoryID, err = uuid.FromString(input.Body.CategoryID)
	if err != nil {
		return uuid.Nil, uuid.Nil, decimal.Decimal{}, "", time.Time{}, huma.NewError(http.StatusBadRequest, "invalid categoryID", err)
	}

	amount, err = decimal.NewFromString(input.Body.Amount)
	if err != nil {
		return uuid.Nil, uuid.Nil, decimal.Decimal{}, "", time.Time{}, huma.NewError(http.StatusBadRequest, "invalid amount", err)
	}

	transactionName = input.Body.TransactionName
	if input.Body.TransactionDate != "" {
		transactionDate, err = time.Parse(time.RFC3339, input.Body.TransactionDate)
		if err != nil {
			return uuid.Nil, uuid.Nil, decimal.Decimal{}, "", time.Time{}, huma.NewError(http.StatusBadRequest, "invalid transactionDate", err)
		}
	}

	return accountID, categoryID, amount, transactionName, transactionDate, nil
}

func (h *CreateTransactionHandler) handle(ctx context.Context, input *CreateTransactionInput) (*CreateTransactionOutput, error) {
	logData := logging.GetLogData(ctx)

	accountID, categoryID, amount, transactionName, transactionDate, err := parseCreateTransactionInput(input)
	if err != nil {
		return nil, err
	}

	if logData != nil {
		logData.AddData("accountID", accountID.String())
		logData.AddData("categoryID", categoryID.String())
	}

	transaction := service.Transaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: transactionName,
		TransactionDate: transactionDate,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("createTransactionMs")
	}
	id, err := h.TransactionService.CreateTransaction(ctx, transaction)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to create transaction", err)
	}

	if logData != nil {
		logData.AddData("transactionID", id.String())
	}

	return &CreateTransactionOutput{
		Status: http.StatusCreated,
		Body:   CreateTransactionResponse{ID: id.String()},
	}, nil
}
