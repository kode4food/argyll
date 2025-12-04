package log

import (
	"log/slog"
	"os"
)

// New constructs a JSON slog.Logger preconfigured at info level
func New(service, env, version string) *slog.Logger {
	return NewWithLevel(service, env, version, slog.LevelInfo)
}

// NewWithLevel constructs a JSON slog.Logger at the provided level
func NewWithLevel(service, env, version string, lvl slog.Level) *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})

	return slog.New(handler).With(
		slog.String("service", service),
		slog.String("env", env),
		slog.String("version", version))
}
