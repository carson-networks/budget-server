package api

import (
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/carson-networks/budget-server/internal/logging"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func LoggingMiddleware(logger *logrus.Logger) func(http.Handler) http.Handler {
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
