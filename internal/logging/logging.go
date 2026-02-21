package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

func SetupLogging() *logrus.Logger {
	logger := logrus.Logger{
		Formatter: &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyLevel: "loglevel",
			},
		},
		Out:   os.Stdout,
		Level: logrus.InfoLevel,
	}

	return &logger
}
