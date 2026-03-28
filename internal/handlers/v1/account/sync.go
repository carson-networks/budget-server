package account

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/gofrs/uuid/v5"

	"github.com/carson-networks/budget-server/internal/sync"
)

// SyncAccountsBody is the request body for POST /v1/accounts/sync.
type SyncAccountsBody struct {
	AccountIDs []string `json:"accountIDs,omitempty" doc:"Optional list of internal account UUIDs to sync. Empty syncs all connected accounts."`
}

// SyncAccountsInput is the Huma input for the sync endpoint.
type SyncAccountsInput struct {
	Body SyncAccountsBody
}

// SyncAccountsOutput is the Huma output for the sync endpoint.
type SyncAccountsOutput struct {
	Body struct{}
}

// SyncAccountsHandler handles POST /v1/accounts/sync.
type SyncAccountsHandler struct {
	Orchestrator *sync.Orchestrator
}

func NewSyncAccountsHandler(orchestrator *sync.Orchestrator) *SyncAccountsHandler {
	return &SyncAccountsHandler{Orchestrator: orchestrator}
}

func (h *SyncAccountsHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "sync-accounts",
		Method:      http.MethodPost,
		Path:        "/v1/accounts/sync",
		Summary:     "Sync accounts",
		Description: "Pulls the latest transactions from external providers for all synced accounts, or for the specified accounts.",
		Tags:        []string{"Accounts"},
	}, h.handle)
}

func (h *SyncAccountsHandler) handle(ctx context.Context, input *SyncAccountsInput) (*SyncAccountsOutput, error) {
	accountUUIDs := make([]uuid.UUID, 0, len(input.Body.AccountIDs))
	for _, idStr := range input.Body.AccountIDs {
		id, err := uuid.FromString(idStr)
		if err != nil {
			return nil, huma.NewError(http.StatusBadRequest, fmt.Sprintf("invalid accountID: %s", idStr))
		}
		accountUUIDs = append(accountUUIDs, id)
	}

	if err := h.Orchestrator.Sync(ctx, accountUUIDs); err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "sync failed", err)
	}

	return &SyncAccountsOutput{}, nil
}
