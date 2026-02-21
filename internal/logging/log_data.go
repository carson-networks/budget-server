package logging

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type LogData struct {
	timeItemsMutex *sync.Mutex
	timeItems      map[string]int64
	dataItems      map[string]interface{}
	logger         *logrus.Logger
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
