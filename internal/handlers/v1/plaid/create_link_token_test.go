package plaid

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ---- mock: linkTokenCreator ----

type mockLinkTokenCreator struct {
	mock.Mock
}

func (m *mockLinkTokenCreator) CreateLinkToken(ctx context.Context) (string, time.Time, error) {
	args := m.Called(ctx)
	return args.String(0), args.Get(1).(time.Time), args.Error(2)
}

// ---- test helpers ----

func newCreateLinkTokenTestAPI(t *testing.T, client linkTokenCreator) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	h := &CreateLinkTokenHandler{PlaidClient: client}
	h.Register(api)
	return api
}

// ---- tests ----

func TestHTTP_CreateLinkToken_Success(t *testing.T) {
	expiration := time.Date(2026, 3, 22, 12, 0, 0, 0, time.UTC)
	token := "link-sandbox-abc123"

	client := &mockLinkTokenCreator{}
	client.On("CreateLinkToken", mock.Anything).Return(token, expiration, nil)

	resp := newCreateLinkTokenTestAPI(t, client).Post("/v1/plaid/link-token", struct{}{})

	assert.Equal(t, http.StatusOK, resp.Code)

	var body CreateLinkTokenOutput
	assert.NoError(t, json.NewDecoder(resp.Body).Decode(&body.Body))
	assert.Equal(t, token, body.Body.LinkToken)
	assert.Equal(t, expiration, body.Body.Expiration)
	client.AssertExpectations(t)
}

func TestHTTP_CreateLinkToken_PlaidError_Returns500(t *testing.T) {
	client := &mockLinkTokenCreator{}
	client.On("CreateLinkToken", mock.Anything).Return("", time.Time{}, errors.New("plaid unavailable"))

	resp := newCreateLinkTokenTestAPI(t, client).Post("/v1/plaid/link-token", struct{}{})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	client.AssertExpectations(t)
}
