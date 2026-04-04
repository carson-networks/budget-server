package api

import (
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/internal/handlers/v1/account"
	"github.com/carson-networks/budget-server/internal/handlers/v1/budget"
	"github.com/carson-networks/budget-server/internal/handlers/v1/category"
	plaidhandler "github.com/carson-networks/budget-server/internal/handlers/v1/plaid"
	"github.com/carson-networks/budget-server/internal/handlers/v1/status"
	"github.com/carson-networks/budget-server/internal/handlers/v1/transaction"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/operator"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	budgetsync "github.com/carson-networks/budget-server/internal/sync"
)

type Rest struct {
	Logger       *logrus.Logger
	Port         string
	Storage      *storage.Storage
	Operator     *operator.OperatorDelegator
	PlaidClient  *plaidclient.Client
	Orchestrator *budgetsync.Orchestrator
}

func (r *Rest) Serve() {
	mux := http.NewServeMux()

	config := huma.DefaultConfig("Budget API", "1.0.0")
	api := humago.New(mux, config)

	statusHandler := status.NewHandler(r.Operator)
	mux.HandleFunc("/status", logging.LoggingWrapper("Status", r.Logger, statusHandler.Handler))

	listTransactionsHandler := transaction.NewListTransactionsHandler(r.Storage.Read().Transactions)
	listTransactionsHandler.Register(api)

	transactionTotalsHandler := transaction.NewTransactionTotalsHandler(r.Storage.Read().Transactions)
	transactionTotalsHandler.Register(api)

	listAccountsHandler := account.NewListAccountsHandler(r.Storage.Read().Accounts)
	listAccountsHandler.Register(api)

	createAccountHandler := account.NewCreateAccountHandler(r.Operator)
	createAccountHandler.Register(api)

	createTransactionHandler := transaction.NewCreateTransactionHandler(r.Operator)
	createTransactionHandler.Register(api)

	listCategoriesHandler := category.NewListCategoriesHandler(r.Storage.Read().Categories)
	listCategoriesHandler.Register(api)

	createCategoryHandler := category.NewCreateCategoryHandler(r.Operator, r.Storage.Read().Categories)
	createCategoryHandler.Register(api)

	updateCategoryHandler := category.NewUpdateCategoryHandler(r.Operator, r.Storage.Read().Categories)
	updateCategoryHandler.Register(api)

	listBudgetsHandler := budget.NewListBudgetsHandler(r.Storage.Read().Budgets)
	listBudgetsHandler.Register(api)

	setBudgetHandler := budget.NewSetBudgetHandler(r.Operator)
	setBudgetHandler.Register(api)

	createLinkTokenHandler := plaidhandler.NewCreateLinkTokenHandler(r.PlaidClient)
	createLinkTokenHandler.Register(api)

	exchangeTokenHandler := plaidhandler.NewExchangeTokenHandler(r.Operator, r.PlaidClient, r.Orchestrator)
	exchangeTokenHandler.Register(api)

	syncAccountsHandler := account.NewSyncAccountsHandler(r.Orchestrator)
	syncAccountsHandler.Register(api)

	handler := LoggingMiddleware(r.Logger)(CorsMiddleware(mux))

	server := http.Server{
		Addr:              ":" + r.Port,
		Handler:           handler,
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
