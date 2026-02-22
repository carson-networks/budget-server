package main

import (
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/api"
	"github.com/carson-networks/budget-server/internal/config"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/service"
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
	svc := service.NewService(dbStorage)

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		httpRest := api.Rest{
			Logger:  logger,
			Port:    "9446",
			Service: svc,
		}
		httpRest.Serve()
	}()

	wg.Wait()
}
