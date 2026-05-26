package haproxy

import (
	"os"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

func testConfig(proxyMode string) *config.Config {
	cfg := &config.Config{}
	cfg.ProxyMode = proxyMode
	cfg.Proxy.Master.Listen = "0.0.0.0"
	cfg.Proxy.Master.Port = 63536
	cfg.Proxy.Master.SocksPort = 63536
	cfg.Proxy.Master.HTTPPort = 63537
	cfg.Proxy.Master.ClientTimeout = 35
	cfg.Proxy.Master.ServerTimeout = 35
	cfg.Proxy.Stats.Listen = "0.0.0.0"
	cfg.Proxy.Stats.Port = 63539
	cfg.Proxy.Stats.URI = "/splitter_status"
	cfg.Proxy.LoadBalanceAlgorithm = "roundrobin"
	cfg.Instances.Retries = 1000
	cfg.HealthCheck.URL = "https://www.google.com/"
	cfg.HealthCheck.Interval = 12
	cfg.HealthCheck.MaxFail = 1
	cfg.HealthCheck.MinimumSuccess = 1
	cfg.Privoxy.StartPort = 6999
	cfg.Paths.TempFiles = "/tmp/splitter"
	cfg.HAProxy.BinaryPath = "/usr/sbin/haproxy"
	cfg.HAProxy.ConfigFile = "/tmp/splitter/splitter_master_proxy.cfg"
	return cfg
}

func testInstances(count int) []*tor.Instance {
	instances := make([]*tor.Instance, count)
	countries := []string{"{US}", "{DE}", "{FR}", "{GB}", "{NL}", "{SE}"}
	for i := 0; i < count; i++ {
		inst := &tor.Instance{
			ID:          i,
			Country:     countries[i%len(countries)],
			SocksPort:   4999 + i,
			ControlPort: 5999 + i,
			HTTPPort:    5199 + i,
		}
		instances[i] = inst
	}
	return instances
}

func TestBuildConfigData_NativeMode(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(3)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if len(data.HTTPBackends) != 3 {
		t.Fatalf("HTTPBackends count = %d, want 3", len(data.HTTPBackends))
	}
	if len(data.SOCKSBackends) != 3 {
		t.Fatalf("SOCKSBackends count = %d, want 3", len(data.SOCKSBackends))
	}

	for _, b := range data.HTTPBackends {
		if !strings.HasPrefix(b.Name, "tor_http_") {
			t.Errorf("HTTP backend name = %q, want prefix tor_http_", b.Name)
		}
	}
	for _, b := range data.SOCKSBackends {
		if !strings.HasPrefix(b.Name, "tor_socks_") {
			t.Errorf("SOCKS backend name = %q, want prefix tor_socks_", b.Name)
		}
	}

	httpPortSet := make(map[int]bool)
	for _, b := range data.HTTPBackends {
		httpPortSet[b.Port] = true
	}
	for _, inst := range instances {
		if !httpPortSet[inst.HTTPPort] {
			t.Errorf("missing HTTP backend for port %d", inst.HTTPPort)
		}
	}
}

func TestBuildConfigData_LegacyMode(t *testing.T) {
	cfg := testConfig("legacy")
	instances := testInstances(3)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if len(data.HTTPBackends) != 3 {
		t.Fatalf("HTTPBackends count = %d, want 3", len(data.HTTPBackends))
	}

	for _, b := range data.HTTPBackends {
		if !strings.HasPrefix(b.Name, "privoxy_") {
			t.Errorf("HTTP backend name = %q, want prefix privoxy_", b.Name)
		}
	}

	expectedPorts := map[int]bool{6999: true, 7000: true, 7001: true}
	for _, b := range data.HTTPBackends {
		if !expectedPorts[b.Port] {
			t.Errorf("unexpected legacy HTTP backend port %d", b.Port)
		}
	}
}

func TestBuildConfigData_NativeModeNoHTTPPort(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(3)
	instances[1].HTTPPort = 0
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if len(data.HTTPBackends) != 2 {
		t.Fatalf("HTTPBackends count = %d, want 2 (one instance has no HTTPPort)", len(data.HTTPBackends))
	}
}

func TestBuildConfigData_BackendShuffle(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(20)
	password := generateStatsPassword()

	sameCount := 0
	iterations := 10

	firstOrder := buildBackendPortList(cfg, instances, password)

	for i := 0; i < iterations; i++ {
		order := buildBackendPortList(cfg, instances, password)
		if order == firstOrder {
			sameCount++
		}
	}

	if sameCount == iterations {
		t.Errorf("backends were never shuffled across %d iterations, expected at least one different order", iterations)
	}
}

func buildBackendPortList(cfg *config.Config, instances []*tor.Instance, password string) string {
	data := BuildConfigData(cfg, instances, password)
	ports := make([]string, len(data.HTTPBackends))
	for i, b := range data.HTTPBackends {
		ports[i] = strings.TrimSpace(b.Name)
	}
	return strings.Join(ports, ",")
}

func TestBuildConfigData_StatsPassword(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(2)
	password := "testpassword12345"

	data := BuildConfigData(cfg, instances, password)

	if data.StatsPassword != password {
		t.Errorf("StatsPassword = %q, want %q", data.StatsPassword, password)
	}
}

func TestBuildConfigData_TimeoutAndRetries(t *testing.T) {
	cfg := testConfig("native")
	cfg.Proxy.Master.ClientTimeout = 60
	cfg.Proxy.Master.ServerTimeout = 45
	cfg.Instances.Retries = 500
	instances := testInstances(1)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if data.ClientTimeout != 60 {
		t.Errorf("ClientTimeout = %d, want 60", data.ClientTimeout)
	}
	if data.ServerTimeout != 45 {
		t.Errorf("ServerTimeout = %d, want 45", data.ServerTimeout)
	}
	if data.Retries != 500 {
		t.Errorf("Retries = %d, want 500", data.Retries)
	}
}

func TestBuildConfigData_PortsFromConfig(t *testing.T) {
	cfg := testConfig("native")
	cfg.Proxy.Master.SocksPort = 9050
	cfg.Proxy.Master.HTTPPort = 9080
	cfg.Proxy.Stats.Port = 9100
	cfg.Proxy.Stats.Listen = "127.0.0.1"
	instances := testInstances(1)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if data.SOCKSPort != 9050 {
		t.Errorf("SOCKSPort = %d, want 9050", data.SOCKSPort)
	}
	if data.HTTPPort != 9080 {
		t.Errorf("HTTPPort = %d, want 9080", data.HTTPPort)
	}
	if data.StatsPort != 9100 {
		t.Errorf("StatsPort = %d, want 9100", data.StatsPort)
	}
	if data.StatsListen != "127.0.0.1" {
		t.Errorf("StatsListen = %q, want %q", data.StatsListen, "127.0.0.1")
	}
}

func TestBuildConfigData_HealthCheckParams(t *testing.T) {
	cfg := testConfig("native")
	cfg.HealthCheck.Interval = 5
	cfg.HealthCheck.MaxFail = 3
	cfg.HealthCheck.MinimumSuccess = 2
	instances := testInstances(1)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	if data.CheckInterval != 5 {
		t.Errorf("CheckInterval = %d, want 5", data.CheckInterval)
	}
	if data.MaxFail != 3 {
		t.Errorf("MaxFail = %d, want 3", data.MaxFail)
	}
	if data.MinSuccess != 2 {
		t.Errorf("MinSuccess = %d, want 2", data.MinSuccess)
	}
	for _, b := range data.HTTPBackends {
		if b.CheckInterval != 5 {
			t.Errorf("HTTP backend CheckInterval = %d, want 5", b.CheckInterval)
		}
		if b.MaxFail != 3 {
			t.Errorf("HTTP backend MaxFail = %d, want 3", b.MaxFail)
		}
		if b.MinSuccess != 2 {
			t.Errorf("HTTP backend MinSuccess = %d, want 2", b.MinSuccess)
		}
	}
}

func TestBuildConfigData_BalanceAlgorithm(t *testing.T) {
	tests := []struct {
		name      string
		algorithm string
	}{
		{"roundrobin", "roundrobin"},
		{"leastconn", "leastconn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig("native")
			cfg.Proxy.LoadBalanceAlgorithm = tt.algorithm
			instances := testInstances(2)
			password := generateStatsPassword()

			data := BuildConfigData(cfg, instances, password)

			if data.BalanceAlgorithm != tt.algorithm {
				t.Errorf("BalanceAlgorithm = %q, want %q", data.BalanceAlgorithm, tt.algorithm)
			}
		})
	}
}

func TestGenerateConfigTemplate(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(3)
	password := generateStatsPassword()

	data := BuildConfigData(cfg, instances, password)

	tmplBytes, err := os.ReadFile("../../templates/haproxy.cfg.gotmpl")
	if err != nil {
		t.Fatalf("read template: %v", err)
	}

	result, err := RenderConfig(data, string(tmplBytes))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	assertContains(t, result, "bind 0.0.0.0:63537")
	assertContains(t, result, "bind 0.0.0.0:63536")
	assertContains(t, result, "bind 0.0.0.0:63539")
	assertContains(t, result, "balance roundrobin")
	assertContains(t, result, "option tcp-check")
	assertContains(t, result, "tcp-check connect")
	assertContains(t, result, "default_backend tor_http")
	assertContains(t, result, "default_backend tor_socks")
	assertContains(t, result, "timeout client  35s")
	assertContains(t, result, "timeout server  35s")
	assertContains(t, result, "retries 1000")
	assertContains(t, result, "stats auth admin:"+password)
	assertContains(t, result, "stats uri /splitter_status")

	httpCount := strings.Count(result, "127.0.0.1:")
	if httpCount < 3 {
		t.Errorf("expected at least 3 backend server lines, found %d", httpCount)
	}
}

func TestStatsPassword_Random(t *testing.T) {
	pw1 := generateStatsPassword()
	pw2 := generateStatsPassword()

	if pw1 == pw2 {
		t.Errorf("two generated passwords are identical: %q", pw1)
	}
}

func TestStatsPassword_Length(t *testing.T) {
	pw := generateStatsPassword()

	if len(pw) != 16 {
		t.Errorf("password length = %d, want 16", len(pw))
	}
}

func TestStatsPassword_Alphanumeric(t *testing.T) {
	pw := generateStatsPassword()
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	for _, c := range pw {
		if !strings.ContainsRune(charset, c) {
			t.Errorf("password contains non-alphanumeric character: %q", c)
		}
	}
}

func TestNewManager(t *testing.T) {
	cfg := testConfig("native")
	cfg.Paths.TempFiles = t.TempDir()
	cfg.HAProxy.ConfigFile = cfg.Paths.TempFiles + "/haproxy.cfg"
	procMgr := process.NewManager(cfg.Paths.TempFiles)

	mgr := NewManager(cfg, procMgr)

	if mgr.StatsPassword() == "" {
		t.Error("StatsPassword() is empty")
	}
	if len(mgr.StatsPassword()) != 16 {
		t.Errorf("StatsPassword() length = %d, want 16", len(mgr.StatsPassword()))
	}
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", needle, haystack)
	}
}
