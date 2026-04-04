package cli

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/user/splitter/internal/config"
)

func SetupLogger(cfg *config.Config) error {
	if !cfg.Logging.Enabled {
		discard := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
		slog.SetDefault(discard)
		return nil
	}

	level, err := parseLogLevel(cfg.Logging.Level)
	if err != nil {
		return fmt.Errorf("SetupLogger: %w", err)
	}

	opts := &slog.HandlerOptions{Level: level}

	if isContainer() {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, opts)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, opts)))
	}

	return nil
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("parseLogLevel: unknown level %q", s)
	}
}

func isContainer() bool {
	if os.Getenv("TERM") == "dumb" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	return false
}
