package api

import (
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/internal/handlers/v1/status"
	"github.com/carson-networks/budget-server/internal/logging"
	"github.com/carson-networks/budget-server/internal/storage"
)

type Rest struct {
	Logger  *logrus.Logger
	Port    string
	Storage storage.Storage
}

func (r *Rest) Serve() {
	statusHandler := status.NewHandler()

	http.HandleFunc("/status", logging.LoggingWrapper("Status", r.Logger, statusHandler.Handler))

	server := http.Server{
		Addr:              ":" + r.Port,
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
