package api

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/account"
	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget"
	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/category"
	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid"
	"github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/account"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/budget"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/category"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/plaid"
	"github.com/carson-networks/budget-server/internal/connecthandlers/v1/transaction"
	"github.com/carson-networks/budget-server/internal/operator"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	budgetsync "github.com/carson-networks/budget-server/internal/sync"
)

// ConnectServer serves ConnectRPC (protobuf + JSON) on a dedicated port alongside the REST API.
type ConnectServer struct {
	Logger       *logrus.Logger
	Port         string
	Storage      *storage.Storage
	Operator     *operator.OperatorDelegator
	PlaidClient  *plaidclient.Client
	Orchestrator *budgetsync.Orchestrator
}

// Serve starts the Connect HTTP server until it exits.
func (c *ConnectServer) Serve() {
	deps := connecthandlers.Deps{
		Storage:      c.Storage,
		Operator:     c.Operator,
		PlaidClient:  c.PlaidClient,
		Orchestrator: c.Orchestrator,
	}

	accountSvc := &v1Account.Service{Deps: deps}
	transactionSvc := &v1Transaction.Service{Deps: deps}
	categorySvc := &v1Category.Service{Deps: deps}
	budgetSvc := &v1Budget.Service{Deps: deps}
	plaidSvc := &v1Plaid.Service{Deps: deps}

	mux := http.NewServeMux()
	for _, reg := range []func() (string, http.Handler){
		func() (string, http.Handler) { return account.NewAccountServiceHandler(accountSvc) },
		func() (string, http.Handler) { return transaction.NewTransactionServiceHandler(transactionSvc) },
		func() (string, http.Handler) { return category.NewCategoryServiceHandler(categorySvc) },
		func() (string, http.Handler) { return budget.NewBudgetServiceHandler(budgetSvc) },
		func() (string, http.Handler) { return plaid.NewPlaidServiceHandler(plaidSvc) },
	} {
		path, h := reg()
		mux.Handle(path, h)
	}

	handler := LoggingMiddleware(c.Logger)(CorsMiddleware(mux))
	server := http.Server{
		Addr:              ":" + c.Port,
		Handler:           handler,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       10 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
	}

	c.Logger.WithField("port", c.Port).Info("ConnectServer.Serve.listening")
	if err := server.ListenAndServe(); err != nil {
		c.Logger.WithError(err).Error("ConnectServer.Serve.listen error")
	}
	c.Logger.Info("ConnectServer.Serve.shutting down")
}
