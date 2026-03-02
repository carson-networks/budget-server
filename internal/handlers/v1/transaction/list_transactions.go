package transaction

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/storage/transaction"
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

// transactionReader is the interface for listing transactions.
type transactionReader interface {
	List(ctx context.Context, filter *transaction.TransactionFilter) (*transaction.TransactionListResult, error)
}

// ListTransactionsHandler handles POST /v1/transaction/list.
type ListTransactionsHandler struct {
	TransactionReader transactionReader
}

// NewListTransactionsHandler creates a new ListTransactionsHandler.
func NewListTransactionsHandler(reader transactionReader) *ListTransactionsHandler {
	return &ListTransactionsHandler{TransactionReader: reader}
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
// Without a cursor, the reader uses its default limit.
func parseListTransactionsInput(input *ListTransactionsInput) (*transaction.TransactionFilter, error) {
	limit := 20
	offset := 0
	var maxCreationTime *time.Time

	if input.Body.Cursor != nil {
		if input.Body.Cursor.Position < 0 {
			return nil, huma.NewError(http.StatusBadRequest, "cursor position must be non-negative")
		}
		offset = input.Body.Cursor.Position
		if input.Body.Cursor.Limit > 0 {
			limit = input.Body.Cursor.Limit
		}
		if input.Body.Cursor.MaxCreationTime != "" {
			parsed, parseErr := time.Parse(time.RFC3339, input.Body.Cursor.MaxCreationTime)
			if parseErr != nil {
				return nil, huma.NewError(http.StatusBadRequest, "invalid cursor maxCreationTime", parseErr)
			}
			maxCreationTime = &parsed
		}
	}

	return &transaction.TransactionFilter{
		Limit:           limit,
		Offset:          offset,
		MaxCreationTime: maxCreationTime,
	}, nil
}

func (h *ListTransactionsHandler) handle(ctx context.Context, input *ListTransactionsInput) (*ListTransactionsOutput, error) {
	logData := logging.GetLogData(ctx)
	filter, err := parseListTransactionsInput(input)
	if err != nil {
		return nil, err
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listTransactionsMs")
	}
	result, err := h.TransactionReader.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list transactions", err)
	}

	transactions := result.Transactions
	if transactions == nil {
		transactions = []*transaction.Transaction{}
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

	if result.NextCursor != nil {
		resp.NextCursor = &ListTransactionsCursor{
			Position:        result.NextCursor.Position,
			Limit:           result.NextCursor.Limit,
			MaxCreationTime: result.NextCursor.MaxCreationTime.Format(time.RFC3339),
		}
	}

	return &ListTransactionsOutput{Body: resp}, nil
}
