package main

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/api"
	"github.com/carson-networks/budget-server/internal/config"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/operator"
	"github.com/carson-networks/budget-server/internal/storage"
)

func main() {
	logger := logging.SetupLogging()
	logrus.Info("budget-server starting")

	envConfig, err := config.ProcessEnvironmentVariables()
	if err != nil {
		logrus.WithError(err).Fatal("config.ProcessEnvironmentVariables")
		return
	}

	dbStorage := storage.NewStorage(envConfig)

	op := operator.NewOperatorDelegator(dbStorage, 4)
	op.Start()
	defer op.Stop()

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		httpRest := api.Rest{
			Logger:   logger,
			Port:     "9446",
			Storage:  dbStorage,
			Operator: op,
		}
		httpRest.Serve()
	}()

	wg.Wait()
}
