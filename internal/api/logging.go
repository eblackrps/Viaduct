package api

import (
	"log/slog"
	"os"
	"strings"
)

var packageLogger = newPackageLogger()

func newPackageLogger() *slog.Logger {
	options := &slog.HandlerOptions{Level: slog.LevelInfo}
	if strings.EqualFold(strings.TrimSpace(os.Getenv("VIADUCT_LOG_FORMAT")), "json") {
		return slog.New(slog.NewJSONHandler(os.Stdout, options))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, options))
}
