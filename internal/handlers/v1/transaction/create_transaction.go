package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"
	"github.com/shopspring/decimal"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
)

// CreateTransactionBody is the request body for creating a transaction.
type CreateTransactionBody struct {
	AccountID       string `json:"accountID" required:"true" doc:"Account UUID"`
	CategoryID      string `json:"categoryID" required:"true" doc:"Category UUID"`
	Amount          string `json:"amount" required:"true" doc:"Decimal amount"`
	TransactionName string `json:"transactionName" required:"true" doc:"Name of the transaction"`
	TransactionDate string `json:"transactionDate" doc:"RFC3339 transaction date, defaults to now"`
}

// CreateTransactionInput is the Huma input for creating a transaction.
type CreateTransactionInput struct {
	Body CreateTransactionBody
}

// CreateTransactionOutput is the Huma output for creating a transaction.
type CreateTransactionOutput struct {
	Status int `json:"status" doc:"HTTP status"`
}

// CreateTransactionHandler handles POST /v1/transaction.
type CreateTransactionHandler struct {
	Operator *operator.OperatorDelegator
}

// NewCreateTransactionHandler creates a new CreateTransactionHandler.
func NewCreateTransactionHandler(op *operator.OperatorDelegator) *CreateTransactionHandler {
	return &CreateTransactionHandler{Operator: op}
}

// Register registers the create transaction endpoint with the Huma API.
func (h *CreateTransactionHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-transaction",
		Method:      http.MethodPost,
		Path:        "/v1/transaction",
		Summary:     "Create transaction",
		Description: "Creates a new transaction.",
		Tags:        []string{"Transactions"},
	}, h.handle)
}

func (h *CreateTransactionHandler) handle(ctx context.Context, input *CreateTransactionInput) (*CreateTransactionOutput, error) {
	accountID, err := uuid.FromString(input.Body.AccountID)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid accountID", err)
	}
	categoryID, err := uuid.FromString(input.Body.CategoryID)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid categoryID", err)
	}
	amount, err := decimal.NewFromString(input.Body.Amount)
	if err != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid amount", err)
	}

	var transactionDate time.Time
	if input.Body.TransactionDate != "" {
		transactionDate, err = time.Parse(time.RFC3339, input.Body.TransactionDate)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, "invalid transactionDate", err)
		}
	} else {
		transactionDate = time.Now()
	}

	action := &actions.CreateTransaction{
		AccountID:       accountID,
		CategoryID:      categoryID,
		Amount:          amount,
		TransactionName: input.Body.TransactionName,
		TransactionDate: transactionDate,
	}

	if err := h.Operator.Process(ctx, action); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to create transaction", err)
	}

	return &CreateTransactionOutput{Status: http.StatusCreated}, nil
}
