package config

import "go.uber.org/zap"

// runtimeContext contains process global settings like the logger,
type runtimeContext struct {
	logger *zap.Logger
}

func (r *runtimeContext) Logger() *zap.Logger {
	return r.logger
}

var ProcessContext runtimeContext

func SetProcessContext(logger *zap.Logger) {
	ProcessContext = runtimeContext{
		logger: logger,
	}
}
