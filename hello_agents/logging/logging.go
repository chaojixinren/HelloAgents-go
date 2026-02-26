package logging

import (
	"log/slog"
	"os"
)

var logger *slog.Logger

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func SetLogger(l *slog.Logger) {
	if l != nil {
		logger = l
	}
}

func Logger() *slog.Logger {
	return logger
}

func Info(msg string, args ...any) {
	logger.Info(msg, args...)
}

func Warn(msg string, args ...any) {
	logger.Warn(msg, args...)
}

func Error(msg string, args ...any) {
	logger.Error(msg, args...)
}

func Debug(msg string, args ...any) {
	logger.Debug(msg, args...)
}
