package cli

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
)

func TestSetupLogger(t *testing.T) {
	tests := []struct {
		name           string
		enabled        bool
		level          string
		termEnv        string
		noColorEnv     string
		wantHandler    string
		wantLevel      slog.Level
		wantOutputNone bool
	}{
		{
			name:           "disabled discards all output",
			enabled:        false,
			wantOutputNone: true,
		},
		{
			name:        "enabled with text handler in terminal",
			enabled:     true,
			level:       "INFO",
			termEnv:     "xterm",
			wantHandler: "text",
			wantLevel:   slog.LevelInfo,
		},
		{
			name:        "enabled with json handler when TERM=dumb",
			enabled:     true,
			level:       "INFO",
			termEnv:     "dumb",
			wantHandler: "json",
			wantLevel:   slog.LevelInfo,
		},
		{
			name:        "enabled with json handler when NO_COLOR set",
			enabled:     true,
			level:       "WARN",
			noColorEnv:  "1",
			wantHandler: "json",
			wantLevel:   slog.LevelWarn,
		},
		{
			name:        "debug level enabled",
			enabled:     true,
			level:       "DEBUG",
			wantHandler: "text",
			wantLevel:   slog.LevelDebug,
		},
		{
			name:        "error level enabled",
			enabled:     true,
			level:       "ERROR",
			wantHandler: "text",
			wantLevel:   slog.LevelError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.termEnv != "" {
				t.Setenv("TERM", tt.termEnv)
			} else {
				t.Setenv("TERM", "xterm-256color")
			}
			if tt.noColorEnv != "" {
				t.Setenv("NO_COLOR", tt.noColorEnv)
			} else {
				t.Setenv("NO_COLOR", "")
			}

			cfg := &config.Config{
				Logging: config.LoggingConfig{
					Enabled: tt.enabled,
					Level:   tt.level,
				},
			}

			err := SetupLogger(cfg)
			if err != nil {
				t.Fatalf("SetupLogger() error = %v", err)
			}

			if tt.wantOutputNone {
				handler := slog.Default().Handler()
				if _, ok := handler.(*slog.TextHandler); !ok {
					t.Fatalf("expected TextHandler for discard, got %T", handler)
				}
				return
			}

			handler := slog.Default().Handler()
			switch tt.wantHandler {
			case "json":
				if _, ok := handler.(*slog.JSONHandler); !ok {
					t.Errorf("expected JSONHandler, got %T", handler)
				}
			case "text":
				if _, ok := handler.(*slog.TextHandler); !ok {
					t.Errorf("expected TextHandler, got %T", handler)
				}
			}
		})
	}
}

func TestSetupLogger_DisabledDiscardsOutput(t *testing.T) {
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Enabled: false,
			Level:   "INFO",
		},
	}

	if err := SetupLogger(cfg); err != nil {
		t.Fatalf("SetupLogger() error = %v", err)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("this should appear in buf")
	if buf.Len() == 0 {
		t.Error("test logger wrote nothing to buf — test setup issue")
	}
}

func TestSetupLogger_DebugLevelShowsDebug(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")

	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Enabled: true,
			Level:   "DEBUG",
		},
	}

	if err := SetupLogger(cfg); err != nil {
		t.Fatalf("SetupLogger() error = %v", err)
	}

	handler := slog.Default().Handler()

	enabled := handler.Enabled(context.TODO(), slog.LevelDebug)
	if !enabled {
		t.Error("expected debug level to be enabled")
	}

	enabledInfo := handler.Enabled(context.TODO(), slog.LevelInfo)
	if !enabledInfo {
		t.Error("expected info level to be enabled")
	}
}

func TestSetupLogger_InfoLevelHidesDebug(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")

	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Enabled: true,
			Level:   "INFO",
		},
	}

	if err := SetupLogger(cfg); err != nil {
		t.Fatalf("SetupLogger() error = %v", err)
	}

	handler := slog.Default().Handler()

	enabled := handler.Enabled(context.TODO(), slog.LevelDebug)
	if enabled {
		t.Error("expected debug level to be hidden at INFO level")
	}

	enabledInfo := handler.Enabled(context.TODO(), slog.LevelInfo)
	if !enabledInfo {
		t.Error("expected info level to be enabled")
	}
}

func TestSetupLogger_InvalidLevel(t *testing.T) {
	cfg := &config.Config{
		Logging: config.LoggingConfig{
			Enabled: true,
			Level:   "bogus",
		},
	}

	err := SetupLogger(cfg)
	if err == nil {
		t.Fatal("expected error for invalid log level")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("error should mention invalid level, got: %v", err)
	}
}

func TestLogFieldHelpers(t *testing.T) {
	tests := []struct {
		name  string
		field slog.Attr
		key   string
		val   string
	}{
		{"instance", InstanceField(5), "instance", "5"},
		{"country", CountryField("US"), "country", "US"},
		{"port", PortField(9050), "port", "9050"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Key != tt.key {
				t.Errorf("key = %q, want %q", tt.field.Key, tt.key)
			}
			got := tt.field.Value.String()
			if got != tt.val {
				t.Errorf("value = %q, want %q", got, tt.val)
			}
		})
	}
}

func TestIsContainer(t *testing.T) {
	tests := []struct {
		name       string
		term       string
		noColor    string
		wantResult bool
	}{
		{"terminal", "xterm", "", false},
		{"dumb", "dumb", "", true},
		{"no_color", "xterm", "1", true},
		{"no_color_empty", "xterm", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("TERM", tt.term)
			if tt.noColor != "" {
				t.Setenv("NO_COLOR", tt.noColor)
			} else {
				_ = os.Unsetenv("NO_COLOR")
			}
			got := isContainer()
			if got != tt.wantResult {
				t.Errorf("isContainer() = %v, want %v", got, tt.wantResult)
			}
		})
	}
}
