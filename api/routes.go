package api

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/internal/handlers/v1/status"
	"github.com/carson-networks/budget-server/internal/handlers/v1/transaction"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/service"
)

type Rest struct {
	Logger  *logrus.Logger
	Port    string
	Service *service.Service
}

func (r *Rest) Serve() {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("Budget API", "1.0.0")
	api := humago.New(mux, config)

	statusHandler := status.NewHandler()
	mux.HandleFunc("/status", logging.LoggingWrapper("Status", r.Logger, statusHandler.Handler))

	createTransactionHandler := transaction.NewCreateTransactionHandler(r.Service.Transaction)
	createTransactionHandler.Register(api)

	listTransactionsHandler := transaction.NewListTransactionsHandler(r.Service.Transaction)
	listTransactionsHandler.Register(api)

	server := http.Server{
		Addr:              ":" + r.Port,
		Handler:           mux,
		ReadTimeout:       time.Duration(30) * time.Second,
		WriteTimeout:      time.Duration(30) * time.Second,
		IdleTimeout:       time.Duration(10) * time.Second,
		ReadHeaderTimeout: time.Duration(10) * time.Second,
	}

	r.Logger.Info("HttpServer.Serve.listening")
	err := server.ListenAndServe()
	if err != nil {
		r.Logger.WithError(err).Error("HttpServer.Serve.listen error")
	}
	r.Logger.Info("HttpServer.Serve.shutting down")
}
