package api

import (
	"net/http"
	"time"

	"connectrpc.com/connect/v2"
	"connectrpc.com/connect/v2/connecthttp"
	"github.com/sirupsen/logrus"

	account "github.com/carson-networks/budget-server/internal/connecthandlers/gen/account/v1"
	budget "github.com/carson-networks/budget-server/internal/connecthandlers/gen/budget/v1"
	category "github.com/carson-networks/budget-server/internal/connecthandlers/gen/category/v1"
	plaid "github.com/carson-networks/budget-server/internal/connecthandlers/gen/plaid/v1"
	transaction "github.com/carson-networks/budget-server/internal/connecthandlers/gen/transaction/v1"
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

	connectServer := connect.NewServer()
	account.RegisterAccountServiceHandler(connectServer, accountSvc)
	transaction.RegisterTransactionServiceHandler(connectServer, transactionSvc)
	category.RegisterCategoryServiceHandler(connectServer, categorySvc)
	budget.RegisterBudgetServiceHandler(connectServer, budgetSvc)
	plaid.RegisterPlaidServiceHandler(connectServer, plaidSvc)

	mux := http.NewServeMux()
	connecthttp.Mount(mux, connectServer)

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
