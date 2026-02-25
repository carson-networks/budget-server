package logging

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type contextKey string

const logDataKey contextKey = "logData"

type LogData struct {
	timeItemsMutex *sync.Mutex
	timeItems      map[string]int64
	dataItems      map[string]interface{}
	logger         *logrus.Logger
}

func WithLogData(ctx context.Context, logData *LogData) context.Context {
	return context.WithValue(ctx, logDataKey, logData)
}

func GetLogData(ctx context.Context) *LogData {
	logData, ok := ctx.Value(logDataKey).(*LogData)
	if !ok {
		return nil
	}
	return logData
}

func NewLogData(logger *logrus.Logger) *LogData {
	return &LogData{
		timeItemsMutex: &sync.Mutex{},
		timeItems:      make(map[string]int64),
		dataItems:      make(map[string]interface{}),
		logger:         logger,
	}
}

func (l *LogData) AddTiming(entryName string) func() {
	startTime := time.Now()

	return func() {
		timeSince := time.Since(startTime).Milliseconds()
		l.timeItemsMutex.Lock()
		defer l.timeItemsMutex.Unlock()
		l.timeItems[entryName] = timeSince
	}
}

func (l *LogData) AddToExistingTiming(entryName string) func() {
	startTime := time.Now()

	return func() {
		timeSince := time.Since(startTime).Milliseconds()
		l.timeItemsMutex.Lock()
		defer l.timeItemsMutex.Unlock()
		l.timeItems[entryName] += timeSince
	}
}

func (l *LogData) AddData(key string, value interface{}) {
	l.dataItems[key] = value
}

func (l *LogData) Log() *logrus.Entry {
	entry := logrus.NewEntry(l.logger)

	for key, value := range l.dataItems {
		entry = entry.WithField(key, value)
	}

	for key, value := range l.timeItems {
		entry = entry.WithField(key, value)
	}

	return entry
}
