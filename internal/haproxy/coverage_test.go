package haproxy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

func TestStatsPassword_NonEmpty(t *testing.T) {
	pw := generateStatsPassword()
	if pw == "" {
		t.Error("generateStatsPassword() returned empty string")
	}
}

func TestStatsPassword_MinLength(t *testing.T) {
	pw := generateStatsPassword()
	if len(pw) < 8 {
		t.Errorf("password length = %d, want >= 8", len(pw))
	}
}

func TestStatsPassword_Unique(t *testing.T) {
	pw1 := generateStatsPassword()
	pw2 := generateStatsPassword()
	if pw1 == pw2 {
		t.Errorf("two calls returned same password: %q", pw1)
	}
}

func TestBuildConfigData_EmptyInstances(t *testing.T) {
	cfg := config.Defaults()
	data := BuildConfigData(cfg, []*tor.Instance{}, "pw")
	if len(data.HTTPBackends) != 0 {
		t.Errorf("HTTPBackends = %d, want 0 for empty instances", len(data.HTTPBackends))
	}
	if len(data.SOCKSBackends) != 0 {
		t.Errorf("SOCKSBackends = %d, want 0 for empty instances", len(data.SOCKSBackends))
	}
	if data.StatsPassword != "pw" {
		t.Errorf("StatsPassword = %q, want %q", data.StatsPassword, "pw")
	}
}

func TestBuildConfigData_NativeModeWithDefaults(t *testing.T) {
	cfg := config.Defaults()
	instances := testInstances(3)
	data := BuildConfigData(cfg, instances, "testpw")

	if len(data.HTTPBackends) != 3 {
		t.Fatalf("HTTPBackends = %d, want 3", len(data.HTTPBackends))
	}
	for _, b := range data.HTTPBackends {
		if !strings.HasPrefix(b.Name, "tor_http_") {
			t.Errorf("native HTTP backend name = %q, want prefix tor_http_", b.Name)
		}
		if b.Address != "127.0.0.1" {
			t.Errorf("native HTTP backend address = %q, want 127.0.0.1", b.Address)
		}
	}
	if len(data.SOCKSBackends) != 3 {
		t.Fatalf("SOCKSBackends = %d, want 3", len(data.SOCKSBackends))
	}
	for _, b := range data.SOCKSBackends {
		if !strings.HasPrefix(b.Name, "tor_socks_") {
			t.Errorf("SOCKS backend name = %q, want prefix tor_socks_", b.Name)
		}
	}
}

func TestBuildConfigData_LegacyModeWithDefaults(t *testing.T) {
	cfg := config.Defaults()
	cfg.ProxyMode = "legacy"
	instances := testInstances(3)
	data := BuildConfigData(cfg, instances, "testpw")

	if len(data.HTTPBackends) != 3 {
		t.Fatalf("HTTPBackends = %d, want 3", len(data.HTTPBackends))
	}
	for _, b := range data.HTTPBackends {
		if !strings.HasPrefix(b.Name, "privoxy_") {
			t.Errorf("legacy HTTP backend name = %q, want prefix privoxy_", b.Name)
		}
	}
	for _, b := range data.HTTPBackends {
		expectedPort := cfg.Privoxy.StartPort + instances[0].ID
		if b.Port < expectedPort || b.Port >= expectedPort+len(instances) {
			t.Errorf("legacy backend port %d outside expected range", b.Port)
		}
	}
}

func TestBuildConfigData_BackendCount(t *testing.T) {
	for _, n := range []int{1, 3, 5, 10} {
		t.Run(fmt.Sprintf("%d_instances", n), func(t *testing.T) {
			cfg := testConfig("native")
			instances := testInstances(n)
			data := BuildConfigData(cfg, instances, "pw")
			if len(data.HTTPBackends) != n {
				t.Errorf("HTTPBackends = %d, want %d", len(data.HTTPBackends), n)
			}
			if len(data.SOCKSBackends) != n {
				t.Errorf("SOCKSBackends = %d, want %d", len(data.SOCKSBackends), n)
			}
		})
	}
}

func TestBuildConfigData_DontProxyRanges(t *testing.T) {
	cfg := config.Defaults()
	if len(cfg.Proxy.DoNotProxy) == 0 {
		t.Skip("DoNotProxy not configured in defaults")
	}
	data := BuildConfigData(cfg, []*tor.Instance{}, "pw")
	if data.Listen == "" {
		t.Error("expected non-empty Listen even with DoNotProxy configured")
	}
	for _, addr := range cfg.Proxy.DoNotProxy {
		if addr == "" {
			t.Error("DoNotProxy entry should not be empty")
		}
	}
}

func TestRenderConfig_ValidTemplate(t *testing.T) {
	cfg := testConfig("native")
	instances := testInstances(2)
	data := BuildConfigData(cfg, instances, "secret123")

	tmplStr := `global
    maxconn 256

defaults
    mode http
    timeout client {{.ClientTimeout}}s
    timeout server {{.ServerTimeout}}s
    retries {{.Retries}}

frontend http_in
    bind {{.Listen}}:{{.HTTPPort}}
    default_backend http_backends

backend http_backends
    balance {{.BalanceAlgorithm}}
{{range .HTTPBackends}}    server {{.Name}} {{.Address}}:{{.Port}} check
{{end}}`

	result, err := RenderConfig(data, tmplStr)
	if err != nil {
		t.Fatalf("RenderConfig() error = %v", err)
	}
	assertContains(t, result, "timeout client 35s")
	assertContains(t, result, "timeout server 35s")
	assertContains(t, result, "retries 1000")
	assertContains(t, result, "balance roundrobin")
	assertContains(t, result, "bind 0.0.0.0:63537")
	for _, b := range data.HTTPBackends {
		assertContains(t, result, fmt.Sprintf("server %s %s:%d", b.Name, b.Address, b.Port))
	}
}

func TestRenderConfig_InvalidTemplate(t *testing.T) {
	data := &ConfigData{ClientTimeout: 30}
	_, err := RenderConfig(data, "{{.InvalidField}}")
	if err == nil {
		t.Error("expected error for invalid template, got nil")
	}
}

func TestNewManager_SetsConfigFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgFile := filepath.Join(tmpDir, "haproxy.cfg")

	cfg := config.Defaults()
	cfg.Paths.TempFiles = tmpDir
	cfg.HAProxy.ConfigFile = cfgFile
	procMgr := process.NewManager(tmpDir)

	mgr := NewManager(cfg, procMgr)
	if mgr.StatsPassword() == "" {
		t.Error("StatsPassword() is empty")
	}

	tmplDir := filepath.Join(tmpDir, "templates")
	if err := os.MkdirAll(tmplDir, 0755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	tmplPath := filepath.Join(tmplDir, "haproxy.cfg.gotmpl")
	minimalTmpl := "test\n"
	if err := os.WriteFile(tmplPath, []byte(minimalTmpl), 0644); err != nil {
		t.Fatalf("write template: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	instances := testInstances(1)
	if err := mgr.GenerateConfig(instances); err != nil {
		t.Fatalf("GenerateConfig() error = %v", err)
	}
	if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
		t.Errorf("config file %q was not created", cfgFile)
	}
}

func TestManager_StopWithoutStart(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Defaults()
	cfg.Paths.TempFiles = tmpDir
	cfg.HAProxy.ConfigFile = filepath.Join(tmpDir, "haproxy.cfg")
	procMgr := process.NewManager(tmpDir)

	mgr := NewManager(cfg, procMgr)
	if err := mgr.Stop(context.TODO()); err != nil {
		t.Errorf("Stop on unstarted manager should return nil, got %v", err)
	}
}
