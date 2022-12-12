package platform

import "github.com/go-logr/logr"

func NewLoggerMock() logr.Logger {
	return logr.New(&LoggerMock{})
}

type LoggerMock struct {
	errors       []error
	infoMessages map[string][]interface{}
}

// Init implements logr.LogSink.
func (log *LoggerMock) Init(logr.RuntimeInfo) {
}

// Info implements logr.InfoLogger.
func (log *LoggerMock) Info(level int, msg string, keysAndValues ...interface{}) {
	if log.infoMessages == nil {
		log.infoMessages = make(map[string][]interface{})
	}

	log.infoMessages[msg] = keysAndValues
}

func (log *LoggerMock) InfoMessages() map[string][]interface{} {
	return log.infoMessages
}

func (log *LoggerMock) ClearInfoMessages() {
	log.infoMessages = nil
}

// Enabled implements logr.InfoLogger.
func (log *LoggerMock) Enabled(level int) bool {
	return true
}

func (log *LoggerMock) Error(err error, msg string, keysAndValues ...interface{}) {
	log.errors = append(log.errors, err)
}

func (log LoggerMock) LastError() error {
	if len(log.errors) == 0 {
		return nil
	}

	return log.errors[len(log.errors)-1]
}

// WithName implements logr.Logger.
func (log *LoggerMock) WithName(_ string) logr.LogSink {
	return log
}

// WithValues implements logr.Logger.
func (log *LoggerMock) WithValues(_ ...interface{}) logr.LogSink {
	return log
}
