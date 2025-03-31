package foxyevent

import (
	"context"
	"log/slog"
)

type SlogLogger struct {
	slogLogger *slog.Logger

	normalLevel slog.Level
	errorLevel  slog.Level
}

func NewSlogLogger(logger *slog.Logger) *SlogLogger {
	return &SlogLogger{
		slogLogger:  logger,
		normalLevel: slog.LevelDebug,
		errorLevel:  slog.LevelError,
	}
}

func (l *SlogLogger) UseErrorLevel(level slog.Level) {
	l.errorLevel = level
}

func (l *SlogLogger) UseLogLevel(level slog.Level) {
	l.normalLevel = level
}

func (l SlogLogger) logEvent(msg string, args ...any) {
	l.slogLogger.Log(context.Background(), l.normalLevel, msg, args...)
}

func (l SlogLogger) logError(msg string, args ...any) {
	l.slogLogger.Log(context.Background(), l.errorLevel, msg, args...)
}

func (l SlogLogger) LogEvent(e Event) {
	switch e := e.(type) {
	case SSEClientConnected:
		l.logEvent("sse client connected", slog.String("client_ip", e.ClientIP))
	case SSEClientDisconnected:
		l.logEvent("sse client disconnected", slog.String("client_ip", e.ClientIP))
	case SSEFailedCreatingEvent:
		l.logError("failed creating sse event", slog.String("err", e.Err.Error()))
	case SSEFailedMarshalEvent:
		l.logError("failed marshalling sse event", slog.String("err", e.Err.Error()))
	case StdioFailedMarhalResponse:
		l.logError("failed marshalling stdio response", slog.String("err", e.Err.Error()))
	case StdioFailedReadingInput:
		l.logError("failed reading stdio input", slog.String("err", e.Err.Error()))
	case StdioSendingResponse:
		l.logEvent("sending stdio response", slog.String("data", string(e.Data)))
	case StdioFailedWriting:
		l.logError("failed writing to stdout", slog.String("err", e.Err.Error()))
	case StreamingHTTPFailedMarshalEvent:
		l.logError("failed marshalling streaming http event", slog.String("err", e.Err.Error()))
	case FailedCreatingSession:
		l.logError("failed creating session", slog.String("err", e.Err.Error()))
	}
}
