package nats

import "go.uber.org/zap"

// Logger define o contrato para logging no pacote nats
// Esta interface é compartilhada por todos os adapters NATS
type Logger interface {
	Info(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
}

// ZapLoggerAdapter adapta zap.Logger para a interface Logger dos adapters NATS
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

func (l *ZapLoggerAdapter) Warn(msg string, fields ...zap.Field) {
	l.logger.Warn(msg, fields...)
}

