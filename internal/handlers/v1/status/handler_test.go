package status

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/carson-networks/budget-server/internal/logging"
)

func createTestLogData() *logging.LogData {
	logger := logging.SetupLogging()
	return logging.NewLogData(logger)
}

func TestHandler_GoodMethod(t *testing.T) {
	statusHandler := NewHandler()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)

	w := httptest.NewRecorder()

	err := statusHandler.Handler(w, req, createTestLogData())
	assert.NoError(t, err)

	res := w.Result()
	assert.Equal(t, 200, res.StatusCode)
}

func TestHandler_BadMethod(t *testing.T) {
	statusHandler := NewHandler()
	req := httptest.NewRequest(http.MethodPost, "/status", nil)
	w := httptest.NewRecorder()

	err := statusHandler.Handler(w, req, createTestLogData())
	assert.Error(t, err)

	res := w.Result()
	assert.Equal(t, 400, res.StatusCode)
}
