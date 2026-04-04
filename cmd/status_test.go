package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/user/splitter/internal/health"
)

func TestNewStatusCmd_Structure(t *testing.T) {
	cmd := newStatusCmd()

	if cmd.Use != "status" {
		t.Errorf("Use = %q, want %q", cmd.Use, "status")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	url, err := cmd.Flags().GetString("status-url")
	if err != nil {
		t.Fatalf("status-url flag error: %v", err)
	}
	if url != "http://localhost:63540/status" {
		t.Errorf("status-url = %q, want %q", url, "http://localhost:63540/status")
	}
}

func TestStatusCmd_WithMockServer(t *testing.T) {
	status := &health.SystemStatus{
		Timestamp:  "2024-01-15 10:30:45",
		TorVersion: "0.4.8",
		Features:   map[string]bool{"conflux": true, "http_tunnel": true, "congestion_control": true, "cgo": false},
		Instances: []health.InstanceStatus{
			{ID: 0, Country: "{US}", State: "ready", SocksPort: 4999, ControlPort: 5999, HTTPPort: 5199},
		},
		TotalInstances:   1,
		ReadyCount:       1,
		FailedCount:      0,
		Processes:        3,
		ProcessBreakdown: map[string]int{"tor": 1, "haproxy": 1},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	}))
	defer srv.Close()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--status-url", srv.URL + "/status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cmd.Execute() error = %v", err)
	}
}

func TestStatusCmd_Unreachable(t *testing.T) {
	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--status-url", "http://localhost:1/status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for unreachable URL, got nil")
	}
	if !strings.Contains(err.Error(), "cannot connect") {
		t.Errorf("error = %q, want to contain 'cannot connect'", err.Error())
	}
	if !strings.Contains(err.Error(), "runStatus") {
		t.Errorf("error = %q, want to contain 'runStatus'", err.Error())
	}
}

func TestStatusCmd_NonOKResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--status-url", srv.URL + "/status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for non-OK response, got nil")
	}
	if !strings.Contains(err.Error(), "unexpected status 500") {
		t.Errorf("error = %q, want to contain 'unexpected status 500'", err.Error())
	}
}

func TestStatusCmd_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()

	cmd := newStatusCmd()
	cmd.SetArgs([]string{"--status-url", srv.URL + "/status"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("error = %q, want to contain 'decode response'", err.Error())
	}
}

func TestRenderStatus(t *testing.T) {
	status := &health.SystemStatus{
		Timestamp:  "2024-01-15 10:30:45",
		TorVersion: "0.4.8",
		Features:   map[string]bool{"conflux": true, "http_tunnel": false},
		Instances: []health.InstanceStatus{
			{ID: 0, Country: "{US}", State: "ready", SocksPort: 4999, ControlPort: 5999, HTTPPort: 5199},
			{ID: 1, Country: "{DE}", State: "failed", SocksPort: 5000, ControlPort: 6000, HTTPPort: 5200},
			{ID: 2, Country: "{FR}", State: "bootstrapping", SocksPort: 5001, ControlPort: 6001, HTTPPort: 5201},
			{ID: 3, Country: "{NL}", State: "starting", SocksPort: 5002, ControlPort: 6002, HTTPPort: 5202},
		},
		TotalInstances:   4,
		ReadyCount:       1,
		FailedCount:      1,
		Processes:        5,
		ProcessBreakdown: map[string]int{"tor": 4, "haproxy": 1},
	}

	output := renderStatus(status)

	if !strings.Contains(output, "SPLITTER Status") {
		t.Error("output should contain 'SPLITTER Status'")
	}
	if !strings.Contains(output, "2024-01-15 10:30:45") {
		t.Error("output should contain timestamp")
	}
	if !strings.Contains(output, "0.4.8") {
		t.Error("output should contain version")
	}
	if !strings.Contains(output, "4 total") {
		t.Error("output should contain total count")
	}
	if !strings.Contains(output, "1 ready") {
		t.Error("output should contain ready count")
	}
	if !strings.Contains(output, "1 failed") {
		t.Error("output should contain failed count")
	}
	if !strings.Contains(output, "{US}") {
		t.Error("output should contain US country")
	}
	if !strings.Contains(output, "{DE}") {
		t.Error("output should contain DE country")
	}
	if !strings.Contains(output, "socks:4999") {
		t.Error("output should contain socks port")
	}
	if !strings.Contains(output, "ctrl:5999") {
		t.Error("output should contain control port")
	}
	if !strings.Contains(output, "http:5199") {
		t.Error("output should contain http port")
	}
	if !strings.Contains(output, "tor:4") {
		t.Error("output should contain process breakdown")
	}
	if !strings.Contains(output, "haproxy:1") {
		t.Error("output should contain haproxy in process breakdown")
	}
	if !strings.Contains(output, ansiGreen) {
		t.Error("output should contain green ANSI codes")
	}
	if !strings.Contains(output, ansiRed) {
		t.Error("output should contain red ANSI codes")
	}
	if !strings.Contains(output, ansiYellow) {
		t.Error("output should contain yellow ANSI codes")
	}
	if !strings.Contains(output, ansiGray) {
		t.Error("output should contain gray ANSI codes")
	}
}

func TestRenderStatus_EmptyInstances(t *testing.T) {
	status := &health.SystemStatus{
		Timestamp:      "2024-01-15 10:30:45",
		TorVersion:     "0.4.8",
		Features:       map[string]bool{},
		Instances:      []health.InstanceStatus{},
		TotalInstances: 0,
		ReadyCount:     0,
		FailedCount:    0,
		Processes:      0,
	}

	output := renderStatus(status)

	if !strings.Contains(output, "0 total") {
		t.Error("output should show 0 total")
	}
	if !strings.Contains(output, "0 ready") {
		t.Error("output should show 0 ready")
	}
	if !strings.Contains(output, "0 failed") {
		t.Error("output should show 0 failed")
	}
	if !strings.Contains(output, "Processes: 0") {
		t.Error("output should show 0 processes")
	}
}

func TestRenderStatus_NoVersion(t *testing.T) {
	status := &health.SystemStatus{
		Timestamp: "2024-01-15 10:30:45",
		Features:  map[string]bool{},
	}

	output := renderStatus(status)

	if !strings.Contains(output, "Tor version:") {
		t.Error("output should contain Tor version label")
	}
}

func TestStateIcon(t *testing.T) {
	tests := []struct {
		state     string
		wantIcon  string
		wantColor string
	}{
		{"ready", "●", ansiGreen},
		{"failed", "○", ansiRed},
		{"bootstrapping", "◎", ansiYellow},
		{"starting", "◦", ansiGray},
		{"unknown", "◦", ansiGray},
		{"", "◦", ansiGray},
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			icon, color := stateIcon(tt.state)
			if icon != tt.wantIcon {
				t.Errorf("icon = %q, want %q", icon, tt.wantIcon)
			}
			if color != tt.wantColor {
				t.Errorf("color = %q, want %q", color, tt.wantColor)
			}
		})
	}
}

func TestFormatFeatures(t *testing.T) {
	features := map[string]bool{
		"conflux":            true,
		"http_tunnel":        true,
		"congestion_control": false,
		"cgo":                false,
	}

	output := formatFeatures(features)

	if !strings.Contains(output, "conflux") {
		t.Error("output should contain 'conflux'")
	}
	if !strings.Contains(output, "http_tunnel") {
		t.Error("output should contain 'http_tunnel'")
	}
	if !strings.Contains(output, "congestion_control") {
		t.Error("output should contain 'congestion_control'")
	}
	if !strings.Contains(output, "cgo") {
		t.Error("output should contain 'cgo'")
	}
	if !strings.Contains(output, "✓") {
		t.Error("output should contain checkmark for enabled features")
	}
	if !strings.Contains(output, "✗") {
		t.Error("output should contain X for disabled features")
	}
	if !strings.Contains(output, " | ") {
		t.Error("output should contain pipe separator")
	}
}

func TestFormatFeatures_Empty(t *testing.T) {
	output := formatFeatures(map[string]bool{})

	if !strings.Contains(output, "conflux") {
		t.Error("output should still contain feature names even when empty")
	}
	if !strings.Contains(output, "✗") {
		t.Error("output should show all features as disabled when map is empty")
	}
}

func TestFormatFeatures_AllEnabled(t *testing.T) {
	features := map[string]bool{
		"conflux":            true,
		"http_tunnel":        true,
		"congestion_control": true,
		"cgo":                true,
	}

	output := formatFeatures(features)

	if strings.Contains(output, "✗") {
		t.Error("output should not contain X when all features are enabled")
	}
}
