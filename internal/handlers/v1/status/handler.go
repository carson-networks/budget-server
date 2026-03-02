package status

import (
	"errors"
	"net/http"

	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/operator"
)

type Handler struct {
	Operator *operator.OperatorDelegator
}

func NewHandler(op *operator.OperatorDelegator) Handler {
	return Handler{Operator: op}
}

func (h *Handler) Handler(w http.ResponseWriter, req *http.Request, logData *logging.LogData) error {
	if req.Method != "GET" {
		w.WriteHeader(http.StatusBadRequest)
		return errors.New("status: method not GET")
	}

	w.WriteHeader(http.StatusOK)
	return nil
}
