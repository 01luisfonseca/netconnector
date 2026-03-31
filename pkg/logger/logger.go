package logger

import (
	"log/slog"
	"os"
)

var defaultLogger *slog.Logger

func init() {
	// Standard Text Handler to stdout. In English.
	opts := &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}
	handler := slog.NewTextHandler(os.Stdout, opts)
	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

// Info logs at LevelInfo.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Error logs at LevelError.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Debug logs at LevelDebug.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Warn logs at LevelWarn.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}
