package account

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

// ListAccountsCursor represents a pagination cursor in request query params.
type ListAccountsCursor struct {
	Position int `query:"position" minimum:"0" doc:"Numeric offset position for the next page"`
	Limit    int `query:"limit" minimum:"1" maximum:"100" doc:"Page size"`
}

// ListAccountsInput is the Huma input for listing accounts.
type ListAccountsInput struct {
	Position int `query:"position" minimum:"0" doc:"Offset for pagination"`
	Limit    int `query:"limit" minimum:"1" maximum:"100" doc:"Page size, default 20"`
}

// ListAccountsResponseBody is the response body for listing accounts.
type ListAccountsResponseBody struct {
	Accounts   []Account `json:"accounts" doc:"Page of accounts"`
	NextCursor *struct {
		Position int `json:"position" doc:"Offset for next page"`
		Limit    int `json:"limit" doc:"Page size"`
	} `json:"nextCursor,omitempty" doc:"Cursor to fetch the next page, absent on the last page"`
}

// ListAccountsOutput is the Huma output for listing accounts.
type ListAccountsOutput struct {
	Body ListAccountsResponseBody
}

// accountReader is the interface for listing accounts.
type accountReader interface {
	List(ctx context.Context, filter *account.AccountFilter) (*account.AccountListResult, error)
}

// ListAccountsHandler handles GET /v1/accounts.
type ListAccountsHandler struct {
	AccountReader accountReader
	Operator      *operator.OperatorDelegator
}

// NewListAccountsHandler creates a new ListAccountsHandler.
func NewListAccountsHandler(reader accountReader, op *operator.OperatorDelegator) *ListAccountsHandler {
	return &ListAccountsHandler{AccountReader: reader, Operator: op}
}

// Register registers the list accounts endpoint with the Huma API.
func (h *ListAccountsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "list-accounts",
		Method:      http.MethodGet,
		Path:        "/v1/accounts",
		Summary:     "List accounts",
		Description: "Returns a paginated list of accounts.",
		Tags:        []string{"Accounts"},
	}, h.handle)
}

func (h *ListAccountsHandler) handle(ctx context.Context, input *ListAccountsInput) (*ListAccountsOutput, error) {
	logData := logging.GetLogData(ctx)

	limit := input.Limit
	if limit == 0 {
		limit = 20
	}
	filter := &account.AccountFilter{
		Limit:  limit,
		Offset: input.Position,
	}

	var stopTimer func()
	if logData != nil {
		stopTimer = logData.AddTiming("listAccountsMs")
	}
	result, err := h.AccountReader.List(ctx, filter)
	if stopTimer != nil {
		stopTimer()
	}
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to list accounts", err)
	}

	accounts := result.Accounts
	if accounts == nil {
		accounts = []*account.Account{}
	}

	if logData != nil {
		logData.AddData("accountCount", len(accounts))
	}

	resp := ListAccountsResponseBody{
		Accounts: make([]Account, len(accounts)),
	}

	for i, acc := range accounts {
		resp.Accounts[i] = Account{
			ID:              acc.ID.String(),
			Name:            acc.Name,
			Type:            int(acc.Type),
			SubType:         acc.SubType,
			Balance:         acc.Balance.String(),
			StartingBalance: acc.StartingBalance.String(),
			CreatedAt:       acc.CreatedAt.Format(time.RFC3339),
		}
	}

	if result.NextCursor != nil {
		resp.NextCursor = &struct {
			Position int `json:"position" doc:"Offset for next page"`
			Limit    int `json:"limit" doc:"Page size"`
		}{
			Position: result.NextCursor.Position,
			Limit:    result.NextCursor.Limit,
		}
	}

	return &ListAccountsOutput{Body: resp}, nil
}
