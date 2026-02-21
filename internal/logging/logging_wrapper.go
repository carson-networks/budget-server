package logging

import (
	"net/http"

	"github.com/sirupsen/logrus"
)

func LoggingWrapper(
	loggingName string,
	log *logrus.Logger,
	handler func(http.ResponseWriter, *http.Request, *LogData) error,
) http.HandlerFunc {
	logData := NewLogData(log)

	return func(w http.ResponseWriter, req *http.Request) {
		log.Infof("Handler.%v.Start", loggingName)

		endTimer := logData.AddTiming("duration")
		defer endTimer()
		err := handler(w, req, logData)
		if err != nil {
			logData.Log().WithError(err).Errorf("Handler.%v.Error", loggingName)
			return
		}

		logData.Log().Infof("Handler.%v.Complete", loggingName)
	}
}
