package archiver

import (
	"log/slog"
	"os"
)

func SetupLogging(level string) {
	logLevels := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}

	lvl, ok := logLevels[level]
	if !ok {
		lvl = slog.LevelInfo
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	}))
	slog.SetDefault(logger)
	slog.SetLogLoggerLevel(lvl)
}
