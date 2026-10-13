package logger

import (
	"log/slog"
	"os"
)

func Init(service string, env string) {
	isProd := env == "production" || env == "prod"

	logLevel := slog.LevelDebug
	if isProd {
		logLevel = slog.LevelInfo
	}

	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	var handler slog.Handler = slog.NewTextHandler(os.Stdout, opts)

	if isProd {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}

	baseLogger := slog.New(handler).With(
		"service", service,
		"env", env,
	)

	slog.SetDefault(baseLogger)
}

func Info(msg string, args ...any) {
	slog.Info(msg, args...)
}

func Error(msg string, args ...any) {
	slog.Error(msg, args...)
}

func Warn(msg string, args ...any) {
	slog.Warn(msg, args...)
}

func Debug(msg string, args ...any) {
	slog.Debug(msg, args...)
}
