package status

import (
	"errors"
	"net/http"

	"github.com/carson-networks/budget-server/internal/logging"
)

type Handler struct{}

func NewHandler() Handler {
	return Handler{}
}

func (h *Handler) Handler(w http.ResponseWriter, req *http.Request, logData *logging.LogData) error {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("status: method not GET")
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
