package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/service"
)

// ListTransactionsCursor represents a pagination cursor in request and response bodies.
// It bundles position, limit, and maxCreationTime so subsequent pages use consistent parameters.
type ListTransactionsCursor struct {
	Position        int    `json:"position" minimum:"0" doc:"Numeric offset position for the next page"`
	Limit           int    `json:"limit" minimum:"1" maximum:"100" doc:"Page size used for this cursor"`
	MaxCreationTime string `json:"maxCreationTime" format:"date-time" doc:"Upper bound on created_at locked in from the first page"`
}

// ListTransactionsBody is the request body for listing transactions.
type ListTransactionsBody struct {
	Cursor *ListTransactionsCursor `json:"cursor,omitempty" doc:"Cursor from a previous response to fetch the next page"`
}

// ListTransactionsInput is the Huma input for listing transactions.
type ListTransactionsInput struct {
	Body ListTransactionsBody
}

// ListTransactionsResponseBody is the response body for listing transactions.
type ListTransactionsResponseBody struct {
	Transactions []Transaction           `json:"transactions" doc:"Page of transactions"`
	NextCursor   *ListTransactionsCursor `json:"nextCursor,omitempty" doc:"Cursor to fetch the next page, absent on the last page"`
}

// ListTransactionsOutput is the Huma output for listing transactions.
type ListTransactionsOutput struct {
	Body ListTransactionsResponseBody
}

// transactionLister is the interface for listing transactions.
type transactionLister interface {
	ListTransactions(ctx context.Context, cursor *service.TransactionCursor) ([]service.Transaction, *service.TransactionCursor, error)
}

// ListTransactionsHandler handles POST /v1/transaction/list.
type ListTransactionsHandler struct {
	TransactionService transactionLister
}

// NewListTransactionsHandler creates a new ListTransactionsHandler.
func NewListTransactionsHandler(svc transactionLister) *ListTransactionsHandler {
	return &ListTransactionsHandler{TransactionService: svc}
}

// Register registers the list transactions endpoint with the Huma API.
func (h *ListTransactionsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-transactions",
		Method:      http.MethodPost,
		Path:        "/v1/transaction/list",
		Summary:     "List transactions",
		Description: "Returns a paginated list of transactions using cursor-based pagination.",
		Tags:        []string{"Transactions"},
	}, h.handle)
}

// parseListTransactionsInput parses and validates the API input.
// When a cursor is provided, limit and maxCreationTime come from it.
// Without a cursor, the service uses its default limit.
func parseListTransactionsInput(input *ListTransactionsInput) (cursor *service.TransactionCursor, err error) {
	if input.Body.Cursor == nil {
		return nil, nil
	}

	if input.Body.Cursor.Position < 0 {
		return nil, huma.NewError(http.StatusBadRequest, "cursor position must be non-negative")
	}

	maxCreationTime, parseErr := time.Parse(time.RFC3339, input.Body.Cursor.MaxCreationTime)
	if parseErr != nil {
		return nil, huma.NewError(http.StatusBadRequest, "invalid cursor maxCreationTime", parseErr)
	}

	return &service.TransactionCursor{
		Position:        input.Body.Cursor.Position,
		Limit:           input.Body.Cursor.Limit,
		MaxCreationTime: maxCreationTime,
	}, nil
}

func (h *ListTransactionsHandler) handle(ctx context.Context, input *ListTransactionsInput) (*ListTransactionsOutput, error) {
	logData := logging.GetLogData(ctx)
	requestCursor, err := parseListTransactionsInput(input)
	if err != nil {
		return nil, err
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listTransactionsMs")
	}
	transactions, nextCursor, err := h.TransactionService.ListTransactions(ctx, requestCursor)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list transactions", err)
	}

	if logData != nil {
		logData.AddData("transactionCount", len(transactions))
	}

	resp := ListTransactionsResponseBody{
		Transactions: make([]Transaction, len(transactions)),
	}

	for i, tx := range transactions {
		resp.Transactions[i] = Transaction{
			ID:              tx.ID.String(),
			AccountID:       tx.AccountID.String(),
			CategoryID:      tx.CategoryID.String(),
			Amount:          tx.Amount.String(),
			TransactionName: tx.TransactionName,
			TransactionDate: tx.TransactionDate.Format(time.RFC3339),
			CreatedAt:       tx.CreatedAt.Format(time.RFC3339),
		}
	}

	if nextCursor != nil {
		resp.NextCursor = &ListTransactionsCursor{
			Position:        nextCursor.Position,
			Limit:           nextCursor.Limit,
			MaxCreationTime: nextCursor.MaxCreationTime.Format(time.RFC3339),
		}
	}

	return &ListTransactionsOutput{Body: resp}, nil
}
