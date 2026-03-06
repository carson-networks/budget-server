package account

import (
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

func newCreateAccountTestAPI(t *testing.T, op operator.IProcessor) humatest.TestAPI {
	t.Helper()
	_, api := humatest.New(t)
	NewCreateAccountHandler(op).Register(api)
	return api
}

func TestHTTP_CreateAccount_Success(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			ca, ok := a.(*actions.CreateAccount)
			return ok &&
				ca.Name == "Checking" &&
				ca.Type == account.AccountTypeCash &&
				ca.SubType == "Personal" &&
				ca.StartingBalance.Equal(decimal.NewFromInt(100))
		})).
		Return(nil)

	resp := newCreateAccountTestAPI(t, mockOp).Post("/v1/accounts", CreateAccountBody{
		Name:            "Checking",
		Type:            0, // AccountTypeCash
		SubType:         "Personal",
		StartingBalance: "100",
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateAccount_InvalidStartingBalance_DefaultsToZero(t *testing.T) {
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.MatchedBy(func(a actions.IAction) bool {
			ca, ok := a.(*actions.CreateAccount)
			return ok &&
				ca.Name == "Savings" &&
				ca.StartingBalance.Equal(decimal.Zero)
		})).
		Return(nil)

	resp := newCreateAccountTestAPI(t, mockOp).Post("/v1/accounts", CreateAccountBody{
		Name:            "Savings",
		Type:            4, // AccountTypeAssets
		SubType:         "High Yield",
		StartingBalance: "not-a-number",
	})

	assert.Equal(t, http.StatusCreated, resp.Code)
	mockOp.AssertExpectations(t)
}

func TestHTTP_CreateAccount_ProcessReturnsError(t *testing.T) {
	processErr := errors.New("storage unavailable")
	mockOp := &operator.MockIProcessor{}
	mockOp.EXPECT().
		Process(mock.Anything, mock.Anything).
		Return(processErr)

	resp := newCreateAccountTestAPI(t, mockOp).Post("/v1/accounts", CreateAccountBody{
		Name:            "Checking",
		Type:            0,
		SubType:         "Personal",
		StartingBalance: "0",
	})

	assert.Equal(t, http.StatusInternalServerError, resp.Code)
	mockOp.AssertExpectations(t)
}
