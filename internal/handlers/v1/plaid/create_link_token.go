package plaid

import (
	"context"
	"net/http"
	"time"

	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/danielgtaylor/huma/v2"
)

// linkTokenCreator is the subset of plaid.Client the create-link-token handler needs.
type linkTokenCreator interface {
	CreateLinkToken(ctx context.Context) (string, time.Time, error)
}

// CreateLinkTokenOutput is the Huma output for creating a Plaid link token.
type CreateLinkTokenOutput struct {
	Body struct {
		LinkToken  string    `json:"linkToken" doc:"Plaid link token to initialise the Plaid Link widget"`
		Expiration time.Time `json:"expiration" doc:"When the link token expires"`
	}
}

// CreateLinkTokenHandler handles POST /v1/plaid/link-token.
type CreateLinkTokenHandler struct {
	PlaidClient linkTokenCreator
}

func NewCreateLinkTokenHandler(client *plaidclient.Client) *CreateLinkTokenHandler {
	return &CreateLinkTokenHandler{PlaidClient: client}
}

func (h *CreateLinkTokenHandler) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "create-plaid-link-token",
		Method:      http.MethodPost,
		Path:        "/v1/plaid/link-token",
		Summary:     "Create Plaid link token",
		Description: "Creates a short-lived Plaid link token for use with the Plaid Link widget.",
		Tags:        []string{"Plaid"},
	}, h.handle)
}

func (h *CreateLinkTokenHandler) handle(ctx context.Context, _ *struct{}) (*CreateLinkTokenOutput, error) {
	token, expiration, err := h.PlaidClient.CreateLinkToken(ctx)
	if err != nil {
		return nil, huma.NewError(http.StatusInternalServerError, "failed to create Plaid link token", err)
	}
	out := &CreateLinkTokenOutput{}
	out.Body.LinkToken = token
	out.Body.Expiration = expiration
	return out, nil
}
