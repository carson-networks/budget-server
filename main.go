package main

import (
	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/api"
	"github.com/carson-networks/budget-server/internal/config"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/operator"
	plaidclient "github.com/carson-networks/budget-server/internal/plaid"
	"github.com/carson-networks/budget-server/internal/storage"
	budgetsync "github.com/carson-networks/budget-server/internal/sync"
	"github.com/carson-networks/budget-server/internal/sync/providers"
)

func main() {
	logger := logging.SetupLogging()
	logrus.Info("budget-server starting")

	envConfig, err := config.ProcessEnvironmentVariables()
	if err != nil {
		logrus.WithError(err).Fatal("config.ProcessEnvironmentVariables")
	}

	dbStorage := storage.NewStorage(envConfig)

	op := operator.NewOperatorDelegator(dbStorage, 4)
	op.Start()
	defer op.Stop()

	plaid := plaidclient.NewClient(envConfig.PlaidClientID, envConfig.PlaidSecret, envConfig.PlaidEnv)

	syncRegistry := budgetsync.NewRegistry()
	syncRegistry.Register(providers.NewPlaidProvider(plaid))
	orchestrator := budgetsync.NewOrchestrator(syncRegistry, dbStorage)

	connectSrv := api.ConnectServer{
		Logger:       logger,
		Port:         "9447",
		Storage:      dbStorage,
		Operator:     op,
		PlaidClient:  plaid,
		Orchestrator: orchestrator,
	}
	connectSrv.Serve()
}
