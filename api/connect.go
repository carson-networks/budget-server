package api

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/gen/budget/v1/budgetv1connect"
	connecthandlers "github.com/carson-networks/budget-server/internal/connecthandlers/v1"
	connectaccount "github.com/carson-networks/budget-server/internal/connecthandlers/v1/account"
	connectbudget "github.com/carson-networks/budget-server/internal/connecthandlers/v1/budget"
	connectcategory "github.com/carson-networks/budget-server/internal/connecthandlers/v1/category"
	connectplaid "github.com/carson-networks/budget-server/internal/connecthandlers/v1/plaid"
	connecttransaction "github.com/carson-networks/budget-server/internal/connecthandlers/v1/transaction"
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

	accountSvc := &connectaccount.Service{Deps: deps}
	transactionSvc := &connecttransaction.Service{Deps: deps}
	categorySvc := &connectcategory.Service{Deps: deps}
	budgetSvc := &connectbudget.Service{Deps: deps}
	plaidSvc := &connectplaid.Service{Deps: deps}

	mux := http.NewServeMux()
	for _, reg := range []func() (string, http.Handler){
		func() (string, http.Handler) { return budgetv1connect.NewAccountServiceHandler(accountSvc) },
		func() (string, http.Handler) { return budgetv1connect.NewTransactionServiceHandler(transactionSvc) },
		func() (string, http.Handler) { return budgetv1connect.NewCategoryServiceHandler(categorySvc) },
		func() (string, http.Handler) { return budgetv1connect.NewBudgetServiceHandler(budgetSvc) },
		func() (string, http.Handler) { return budgetv1connect.NewPlaidServiceHandler(plaidSvc) },
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
