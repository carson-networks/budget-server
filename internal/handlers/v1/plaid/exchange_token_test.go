package plaid

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/operator/actions"
	"github.com/carson-networks/budget-server/internal/storage/account"
)

// ---- mock: tokenExchanger ----

type mockTokenExchanger struct {
	mock.Mock
}

func (m *mockTokenExchanger) ExchangePublicToken(ctx context.Context, publicToken string) (string, string, error) {
	args := m.Called(ctx, publicToken)
	return args.String(0), args.String(1), args.Error(2)
}

// ---- test helpers ----

func newExchangeTokenTestAPI(t *testing.T, op operator.IProcessor, client tokenExchanger) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	h := &ExchangeTokenHandler{Operator: op, PlaidClient: client}
	h.Register(api)
	return api
}

// ---- tests ----

func TestHTTP_ExchangeToken_Success(t *testing.T) {
	client := &mockTokenExchanger{}
	client.On("ExchangePublicToken", mock.Anything, "public-token-123").
		Return("access-token-abc", "plaid-item-xyz", nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			lp, ok := a.(*actions.LinkPlaidItem)
			return ok &&
				lp.AccessToken == "access-token-abc" &&
				lp.PlaidItemID == "plaid-item-xyz" &&
				lp.InstitutionID == "ins_1" &&
				lp.InstitutionName == "First Bank" &&
				len(lp.Accounts) == 1 &&
				lp.Accounts[0].PlaidAccountID == "plaid-acc-1" &&
				lp.Accounts[0].Name == "Checking" &&
				lp.Accounts[0].Type == account.AccountTypeCash &&
				lp.Accounts[0].Balance.Equal(decimal.NewFromFloat(500.00))
		})).
		Return(nil)

	resp := newExchangeTokenTestAPI(t, mockOp, client).Post("/v1/plaid/exchange-token", ExchangeTokenBody{
		PublicToken:     "public-token-123",
		InstitutionID:   "ins_1",
		InstitutionName: "First Bank",
		Accounts: []SelectedAccount{
			{PlaidAccountID: "plaid-acc-1", Name: "Checking", Type: 0, SubType: "personal", Balance: decimal.NewFromFloat(500.00)},
		},
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestHTTP_ExchangeToken_NoAccounts_Returns400(t *testing.T) {
	resp := newExchangeTokenTestAPI(t, nil, nil).Post("/v1/plaid/exchange-token", ExchangeTokenBody{
		PublicToken:     "public-token-123",
		InstitutionID:   "ins_1",
		InstitutionName: "First Bank",
		Accounts:        []SelectedAccount{},
	})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestHTTP_ExchangeToken_PlaidError_Returns500(t *testing.T) {
	client := &mockTokenExchanger{}
	client.On("ExchangePublicToken", mock.Anything, "public-token-bad").
		Return("", "", errors.New("plaid exchange failed"))

	resp := newExchangeTokenTestAPI(t, nil, client).Post("/v1/plaid/exchange-token", ExchangeTokenBody{
		PublicToken:     "public-token-bad",
		InstitutionID:   "ins_1",
		InstitutionName: "First Bank",
		Accounts: []SelectedAccount{
			{PlaidAccountID: "plaid-acc-1", Name: "Checking", Type: 0, SubType: "personal"},
		},
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	client.AssertExpectations(t)
}

func TestHTTP_ExchangeToken_OperatorError_Returns500(t *testing.T) {
	client := &mockTokenExchanger{}
	client.On("ExchangePublicToken", mock.Anything, "public-token-123").
		Return("access-token-abc", "plaid-item-xyz", nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().Process(mock.Anything, mock.Anything).Return(errors.New("storage unavailable"))

	resp := newExchangeTokenTestAPI(t, mockOp, client).Post("/v1/plaid/exchange-token", ExchangeTokenBody{
		PublicToken:     "public-token-123",
		InstitutionID:   "ins_1",
		InstitutionName: "First Bank",
		Accounts: []SelectedAccount{
			{PlaidAccountID: "plaid-acc-1", Name: "Savings", Type: 4, SubType: "high-yield"},
		},
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockOp.AssertExpectations(t)
	client.AssertExpectations(t)
}

func TestHTTP_ExchangeToken_MultipleAccounts_AllLinked(t *testing.T) {
	client := &mockTokenExchanger{}
	client.On("ExchangePublicToken", mock.Anything, "pub-tok").
		Return("acc-tok", "item-id", nil)

	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			lp, ok := a.(*actions.LinkPlaidItem)
			return ok && len(lp.Accounts) == 2
		})).
		Return(nil)

	resp := newExchangeTokenTestAPI(t, mockOp, client).Post("/v1/plaid/exchange-token", ExchangeTokenBody{
		PublicToken:     "pub-tok",
		InstitutionID:   "ins_2",
		InstitutionName: "Second Bank",
		Accounts: []SelectedAccount{
			{PlaidAccountID: "p-1", Name: "Checking", Type: 0, SubType: "personal"},
			{PlaidAccountID: "p-2", Name: "Savings", Type: 0, SubType: "personal"},
		},
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
	client.AssertExpectations(t)
}
