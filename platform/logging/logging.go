package logging

import (
	"log/slog"
	"os"
	"strings"
)

func New(level string, format string) *slog.Logger {
	logLevel := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}
	var h slog.Handler
	switch strings.ToLower(format) {
	case "text", "keyvalue", "console":
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	default:
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})
	}
	return slog.New(h)
}
