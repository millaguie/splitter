package haproxy

import (
	"os"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/tor"
)

func haproxyTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.ProxyMode = "native"
	cfg.Proxy.Master.Listen = "0.0.0.0"
	cfg.Proxy.Master.SocksPort = 63536
	cfg.Proxy.Master.HTTPPort = 63537
	cfg.Proxy.Master.ClientTimeout = 35
	cfg.Proxy.Master.ServerTimeout = 35
	cfg.Proxy.Stats.Listen = "0.0.0.0"
	cfg.Proxy.Stats.Port = 63539
	cfg.Proxy.Stats.URI = "/splitter_status"
	cfg.Proxy.LoadBalanceAlgorithm = "roundrobin"
	cfg.Instances.Retries = 3
	cfg.HealthCheck.URL = "https://check.example.com/"
	cfg.HealthCheck.Interval = 10
	cfg.HealthCheck.MaxFail = 2
	cfg.HealthCheck.MinimumSuccess = 1
	cfg.Privoxy.StartPort = 6999
	cfg.Paths.TempFiles = "/tmp/splitter"
	return cfg
}

func haproxyTestInstances(count int) []*tor.Instance {
	countries := []string{"{US}", "{DE}", "{FR}", "{GB}"}
	instances := make([]*tor.Instance, count)
	for i := 0; i < count; i++ {
		instances[i] = &tor.Instance{
			ID:        i,
			Country:   countries[i%len(countries)],
			SocksPort: 4999 + i,
			HTTPPort:  5199 + i,
		}
	}
	return instances
}

func readHAProxyTemplate(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../templates/haproxy.cfg.gotmpl")
	if err != nil {
		t.Fatalf("read haproxy template: %v", err)
	}
	return string(data)
}

func haproxyContains(t *testing.T, result, substr string) {
	t.Helper()
	if !strings.Contains(result, substr) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", substr, result)
	}
}

func haproxyNotContains(t *testing.T, result, substr string) {
	t.Helper()
	if strings.Contains(result, substr) {
		t.Errorf("expected output NOT to contain %q\nfull output:\n%s", substr, result)
	}
}

func TestHAProxyTemplate_ContainsStatsSection(t *testing.T) {
	cfg := haproxyTestConfig()
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "testpw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "stats enable")
	haproxyContains(t, result, "stats uri /splitter_status")
	haproxyContains(t, result, "stats auth admin:testpw")
	haproxyContains(t, result, "stats realm SPLITTER")
}

func TestHAProxyTemplate_ContainsFrontends(t *testing.T) {
	cfg := haproxyTestConfig()
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "frontend http_in")
	haproxyContains(t, result, "frontend socks_in")
	haproxyContains(t, result, "default_backend tor_http")
	haproxyContains(t, result, "default_backend tor_socks")
	haproxyContains(t, result, "bind 0.0.0.0:63537")
	haproxyContains(t, result, "bind 0.0.0.0:63536")
}

func TestHAProxyTemplate_ContainsBackends(t *testing.T) {
	cfg := haproxyTestConfig()
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "backend tor_http")
	haproxyContains(t, result, "backend tor_socks")
}

func TestHAProxyTemplate_BackendServers(t *testing.T) {
	cfg := haproxyTestConfig()
	instances := haproxyTestInstances(2)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	httpServerCount := strings.Count(result, "server tor_http_")
	socksServerCount := strings.Count(result, "server tor_socks_")

	if httpServerCount != 2 {
		t.Errorf("expected 2 HTTP server lines, got %d", httpServerCount)
	}
	if socksServerCount != 2 {
		t.Errorf("expected 2 SOCKS server lines, got %d", socksServerCount)
	}
}

func TestHAProxyTemplate_Timeout(t *testing.T) {
	cfg := haproxyTestConfig()
	cfg.Proxy.Master.ClientTimeout = 60
	cfg.Proxy.Master.ServerTimeout = 45
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "timeout client  60s")
	haproxyContains(t, result, "timeout server  45s")
}

func TestHAProxyTemplate_BalanceAlgorithm(t *testing.T) {
	cfg := haproxyTestConfig()
	cfg.Proxy.LoadBalanceAlgorithm = "roundrobin"
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	balanceCount := strings.Count(result, "balance roundrobin")
	if balanceCount < 2 {
		t.Errorf("expected at least 2 'balance roundrobin' lines (http + socks backend), got %d", balanceCount)
	}
}

func TestHAProxyTemplate_LegacyMode_PrivoxyBackends(t *testing.T) {
	cfg := haproxyTestConfig()
	cfg.ProxyMode = "legacy"
	instances := haproxyTestInstances(2)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	privoxyCount := strings.Count(result, "server privoxy_")
	if privoxyCount != 2 {
		t.Errorf("expected 2 privoxy server lines in legacy mode, got %d", privoxyCount)
	}
}

func TestHAProxyTemplate_LeastconnAlgorithm(t *testing.T) {
	cfg := haproxyTestConfig()
	cfg.Proxy.LoadBalanceAlgorithm = "leastconn"
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	balanceCount := strings.Count(result, "balance leastconn")
	if balanceCount < 2 {
		t.Errorf("expected at least 2 'balance leastconn' lines, got %d", balanceCount)
	}
	haproxyNotContains(t, result, "balance roundrobin")
}

func TestHAProxyTemplate_HealthCheck(t *testing.T) {
	cfg := haproxyTestConfig()
	cfg.HealthCheck.URL = "https://check.example.com/"
	instances := haproxyTestInstances(1)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "option tcp-check")
	haproxyContains(t, result, "tcp-check connect")
}

func TestHAProxyTemplate_EmptyInstances(t *testing.T) {
	cfg := haproxyTestConfig()
	instances := haproxyTestInstances(0)
	data := BuildConfigData(cfg, instances, "pw")

	result, err := RenderConfig(data, readHAProxyTemplate(t))
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}

	haproxyContains(t, result, "backend tor_http")
	haproxyContains(t, result, "backend tor_socks")

	serverLineCount := strings.Count(result, "\n    server ")
	if serverLineCount != 0 {
		t.Errorf("expected 0 server lines with empty instances, got %d", serverLineCount)
	}
}
