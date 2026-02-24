package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

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
	Limit  int                     `json:"limit,omitempty" minimum:"1" maximum:"100" doc:"Page size (1-100, default 20)"`
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
	ListTransactions(ctx context.Context, query service.TransactionListQuery) (*service.TransactionListResult, error)
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

const defaultLimit = 20

// parseListTransactionsInput parses and validates the API input.
// When a cursor is provided, its limit and maxCreationTime override the top-level fields.
func parseListTransactionsInput(input *ListTransactionsInput) (query service.TransactionListQuery, err error) {
	if input.Body.Cursor != nil {
		if input.Body.Cursor.Position < 0 {
			return query, huma.NewError(http.StatusBadRequest, "cursor position must be non-negative")
		}

		maxCreationTime, parseErr := time.Parse(time.RFC3339, input.Body.Cursor.MaxCreationTime)
		if parseErr != nil {
			return query, huma.NewError(http.StatusBadRequest, "invalid cursor maxCreationTime", parseErr)
		}

		query.Limit = input.Body.Cursor.Limit
		query.Cursor = &service.TransactionCursor{
			Position:        input.Body.Cursor.Position,
			Limit:           input.Body.Cursor.Limit,
			MaxCreationTime: maxCreationTime,
		}
		return query, nil
	}

	query.Limit = input.Body.Limit
	if query.Limit == 0 {
		query.Limit = defaultLimit
	}

	return query, nil
}

func (h *ListTransactionsHandler) handle(ctx context.Context, input *ListTransactionsInput) (*ListTransactionsOutput, error) {
	query, err := parseListTransactionsInput(input)
	if err != nil {
		return nil, err
	}

	result, err := h.TransactionService.ListTransactions(ctx, query)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list transactions", err)
	}

	resp := ListTransactionsResponseBody{
		Transactions: make([]Transaction, len(result.Transactions)),
	}

	for i, tx := range result.Transactions {
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

	if result.NextCursor != nil {
		resp.NextCursor = &ListTransactionsCursor{
			Position:        result.NextCursor.Position,
			Limit:           result.NextCursor.Limit,
			MaxCreationTime: result.NextCursor.MaxCreationTime.Format(time.RFC3339),
		}
	}

	return &ListTransactionsOutput{Body: resp}, nil
}
