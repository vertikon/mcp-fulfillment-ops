package redis

import "go.uber.org/zap"

// ZapLoggerAdapter adapta zap.Logger para a interface Logger do adapter Redis
type ZapLoggerAdapter struct {
	logger *zap.Logger
}

// NewZapLoggerAdapter cria um novo adapter
func NewZapLoggerAdapter(logger *zap.Logger) Logger {
	return &ZapLoggerAdapter{logger: logger}
}

func (l *ZapLoggerAdapter) Info(msg string, fields ...zap.Field) {
	l.logger.Info(msg, fields...)
}

func (l *ZapLoggerAdapter) Error(msg string, fields ...zap.Field) {
	l.logger.Error(msg, fields...)
}
