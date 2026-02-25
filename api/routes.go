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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logData := logging.NewLogData(logger)
			logData.AddData("method", r.Method)
			logData.AddData("path", r.URL.Path)

			ctx := logging.WithLogData(r.Context(), logData)
			r = r.WithContext(ctx)

			logger.WithFields(logrus.Fields{
				"method": r.Method,
				"path":   r.URL.Path,
			}).Info("Handler.Start")

			stopTimer := logData.AddTiming("durationMs")
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
			next.ServeHTTP(rw, r)
			stopTimer()

			logData.AddData("status", rw.statusCode)
			logData.Log().Info("Handler.Complete")
		})
	}
}

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

	handler := loggingMiddleware(r.Logger)(corsMiddleware(mux))

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
