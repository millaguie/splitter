package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

func TestCollectStatus(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)
	torMgr.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 8, Release: 0}, []string{"{US}", "{DE}"})

	status := CollectStatus(torMgr, procMgr)

	if status.TorVersion != "0.4.8.0" {
		t.Errorf("TorVersion = %q, want %q", status.TorVersion, "0.4.8.0")
	}
	if status.TotalInstances != 2 {
		t.Errorf("TotalInstances = %d, want 2", status.TotalInstances)
	}
	if len(status.Instances) != 2 {
		t.Fatalf("len(Instances) = %d, want 2", len(status.Instances))
	}
	if status.Instances[0].State != "starting" {
		t.Errorf("Instances[0].State = %q, want %q", status.Instances[0].State, "starting")
	}
	if status.Instances[0].Country != "{US}" {
		t.Errorf("Instances[0].Country = %q, want %q", status.Instances[0].Country, "{US}")
	}
	if status.Instances[0].SocksPort != 4999 {
		t.Errorf("Instances[0].SocksPort = %d, want 4999", status.Instances[0].SocksPort)
	}
	if status.Instances[0].ControlPort != 5999 {
		t.Errorf("Instances[0].ControlPort = %d, want 5999", status.Instances[0].ControlPort)
	}
	if status.Instances[0].HTTPPort != 5199 {
		t.Errorf("Instances[0].HTTPPort = %d, want 5199", status.Instances[0].HTTPPort)
	}
	if status.Instances[1].SocksPort != 5000 {
		t.Errorf("Instances[1].SocksPort = %d, want 5000", status.Instances[1].SocksPort)
	}
	if status.Processes != 0 {
		t.Errorf("Processes = %d, want 0", status.Processes)
	}
	if len(status.ProcessBreakdown) != 0 {
		t.Errorf("ProcessBreakdown = %v, want empty", status.ProcessBreakdown)
	}
}

func TestCollectStatus_Empty(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)

	status := CollectStatus(torMgr, procMgr)

	if status.TorVersion != "" {
		t.Errorf("TorVersion = %q, want empty", status.TorVersion)
	}
	if status.TotalInstances != 0 {
		t.Errorf("TotalInstances = %d, want 0", status.TotalInstances)
	}
	if len(status.Instances) != 0 {
		t.Errorf("len(Instances) = %d, want 0", len(status.Instances))
	}
	if len(status.Features) != 0 {
		t.Errorf("Features = %v, want empty", status.Features)
	}
}

func TestCollectStatus_Features(t *testing.T) {
	tests := []struct {
		name    string
		version *tor.Version
		want    map[string]bool
	}{
		{
			name:    "0.4.8 conflux and http_tunnel",
			version: &tor.Version{Major: 0, Minor: 4, Patch: 8},
			want: map[string]bool{
				"conflux":            true,
				"http_tunnel":        true,
				"congestion_control": true,
				"cgo":                false,
			},
		},
		{
			name:    "0.4.9 all features",
			version: &tor.Version{Major: 0, Minor: 4, Patch: 9},
			want: map[string]bool{
				"conflux":            true,
				"http_tunnel":        true,
				"congestion_control": true,
				"cgo":                true,
			},
		},
		{
			name:    "0.4.6 no features",
			version: &tor.Version{Major: 0, Minor: 4, Patch: 6},
			want: map[string]bool{
				"conflux":            false,
				"http_tunnel":        false,
				"congestion_control": false,
				"cgo":                false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testStatusConfig(t)
			procMgr := process.NewManager(cfg.Paths.TempFiles)
			torMgr := tor.NewManager(cfg, procMgr)
			torMgr.CreateFromVersion(tt.version, []string{"{US}"})

			status := CollectStatus(torMgr, procMgr)

			for key, want := range tt.want {
				if got := status.Features[key]; got != want {
					t.Errorf("Features[%q] = %v, want %v", key, got, want)
				}
			}
		})
	}
}

func TestCollectStatus_JSONRoundTrip(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)
	torMgr.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 8}, []string{"{US}"})

	status := CollectStatus(torMgr, procMgr)

	data, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var decoded SystemStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal error: %v", err)
	}

	if decoded.TorVersion != status.TorVersion {
		t.Errorf("TorVersion = %q, want %q", decoded.TorVersion, status.TorVersion)
	}
	if decoded.TotalInstances != status.TotalInstances {
		t.Errorf("TotalInstances = %d, want %d", decoded.TotalInstances, status.TotalInstances)
	}
	if decoded.ReadyCount != status.ReadyCount {
		t.Errorf("ReadyCount = %d, want %d", decoded.ReadyCount, status.ReadyCount)
	}
	if len(decoded.Instances) != len(status.Instances) {
		t.Fatalf("len(Instances) = %d, want %d", len(decoded.Instances), len(status.Instances))
	}
	if decoded.Instances[0].SocksPort != status.Instances[0].SocksPort {
		t.Errorf("SocksPort = %d, want %d", decoded.Instances[0].SocksPort, status.Instances[0].SocksPort)
	}
	if decoded.Instances[0].Country != status.Instances[0].Country {
		t.Errorf("Country = %q, want %q", decoded.Instances[0].Country, status.Instances[0].Country)
	}
}

func TestCollectStatus_Timestamp(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)

	status := CollectStatus(torMgr, procMgr)

	if status.Timestamp == "" {
		t.Error("Timestamp should not be empty")
	}
	if len(status.Timestamp) != 19 {
		t.Errorf("Timestamp = %q, want format '2006-01-02 15:04:05'", status.Timestamp)
	}
}

func TestStatusHandler(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)
	torMgr.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 8, Release: 0}, []string{"{US}"})

	handler := StatusHandler(torMgr, procMgr)

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Status = %d, want %d", w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}

	var status SystemStatus
	if err := json.NewDecoder(w.Body).Decode(&status); err != nil {
		t.Fatalf("json.Decode error: %v", err)
	}
	if status.TorVersion != "0.4.8.0" {
		t.Errorf("TorVersion = %q, want %q", status.TorVersion, "0.4.8.0")
	}
	if status.TotalInstances != 1 {
		t.Errorf("TotalInstances = %d, want 1", status.TotalInstances)
	}
}

func TestStatusHandler_MultipleRequests(t *testing.T) {
	cfg := testStatusConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	torMgr := tor.NewManager(cfg, procMgr)
	torMgr.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 9, Release: 0}, []string{"{US}", "{DE}"})

	handler := StatusHandler(torMgr, procMgr)

	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/status", nil)
		w := httptest.NewRecorder()
		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("request %d: Status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}
}

func TestCategorizeProcesses(t *testing.T) {
	procs := []*process.Process{
		{Name: "tor-0"},
		{Name: "tor-1"},
		{Name: "tor-2"},
		{Name: "haproxy"},
		{Name: "privoxy-0"},
	}

	result := categorizeProcesses(procs)

	if result["tor"] != 3 {
		t.Errorf("tor count = %d, want 3", result["tor"])
	}
	if result["haproxy"] != 1 {
		t.Errorf("haproxy count = %d, want 1", result["haproxy"])
	}
	if result["privoxy"] != 1 {
		t.Errorf("privoxy count = %d, want 1", result["privoxy"])
	}
}

func TestCategorizeProcesses_Empty(t *testing.T) {
	result := categorizeProcesses(nil)
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}

	result = categorizeProcesses([]*process.Process{})
	if len(result) != 0 {
		t.Errorf("expected empty map, got %v", result)
	}
}

func TestCategorizeProcesses_NoDash(t *testing.T) {
	procs := []*process.Process{
		{Name: "haproxy"},
		{Name: "nginx"},
	}

	result := categorizeProcesses(procs)

	if result["haproxy"] != 1 {
		t.Errorf("haproxy count = %d, want 1", result["haproxy"])
	}
	if result["nginx"] != 1 {
		t.Errorf("nginx count = %d, want 1", result["nginx"])
	}
}

func testStatusConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199
	cfg.Tor.ControlAuth = "cookie"
	cfg.Paths.TempFiles = t.TempDir()
	cfg.Instances.PerCountry = 1
	cfg.Relay.Enforce = "entry"
	return cfg
}
