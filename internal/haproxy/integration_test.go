//go:build integration

package haproxy

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

func TestIntegration_HAProxyConfigValidation(t *testing.T) {
	if _, err := exec.LookPath("haproxy"); err != nil {
		t.Skip("haproxy not available")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.ProxyMode = "native"
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
	cfg.Paths.TempFiles = tmpDir
	cfg.HAProxy.BinaryPath, _ = exec.LookPath("haproxy")
	cfg.HAProxy.ConfigFile = filepath.Join(tmpDir, "haproxy.cfg")

	instances := []*tor.Instance{
		{ID: 0, Country: "{US}", SocksPort: 4999, ControlPort: 5999, HTTPPort: 5199},
		{ID: 1, Country: "{DE}", SocksPort: 5000, ControlPort: 6000, HTTPPort: 5200},
	}

	procMgr := process.NewManager(tmpDir)
	mgr := NewManager(cfg, procMgr)

	if err := mgr.GenerateConfig(instances); err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	cmd := exec.Command(cfg.HAProxy.BinaryPath, "-c", "-f", cfg.HAProxy.ConfigFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("haproxy config validation failed:\n%s\n%s", output, err)
	}
}

func TestIntegration_StartStopHAProxy(t *testing.T) {
	if _, err := exec.LookPath("haproxy"); err != nil {
		t.Skip("haproxy not available")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.ProxyMode = "native"
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
	cfg.Paths.TempFiles = tmpDir
	cfg.HAProxy.BinaryPath, _ = exec.LookPath("haproxy")
	cfg.HAProxy.ConfigFile = filepath.Join(tmpDir, "haproxy.cfg")

	instances := []*tor.Instance{
		{ID: 0, Country: "{US}", SocksPort: 4999, ControlPort: 5999, HTTPPort: 5199},
	}

	procMgr := process.NewManager(tmpDir)
	mgr := NewManager(cfg, procMgr)

	if err := mgr.GenerateConfig(instances); err != nil {
		t.Fatalf("GenerateConfig: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	if err := mgr.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	_ = os.Remove(cfg.HAProxy.ConfigFile)
}
