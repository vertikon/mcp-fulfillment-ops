package app

import "go.uber.org/zap"

// ZapLoggerAdapter adapta zap.Logger para a interface Logger dos casos de uso
type ZapLoggerAdapter struct {
	logger *zap.Logger
}

// NewZapLoggerAdapter cria um novo adapter
func NewZapLoggerAdapter(logger *zap.Logger) Logger {
	return &ZapLoggerAdapter{logger: logger}
}

func (l *ZapLoggerAdapter) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Sugar().Infof(msg, args...)
	} else {
		l.logger.Info(msg)
	}
}

func (l *ZapLoggerAdapter) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Sugar().Errorf(msg, args...)
	} else {
		l.logger.Error(msg)
	}
}

func (l *ZapLoggerAdapter) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		l.logger.Sugar().Warnf(msg, args...)
	} else {
		l.logger.Warn(msg)
	}
}
