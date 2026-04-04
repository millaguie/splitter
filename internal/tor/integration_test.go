//go:build integration

package tor

import (
	"context"
	"testing"
	"time"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

func TestIntegration_DetectVersion(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := DetectVersion(ctx, "/usr/bin/tor")
	if err != nil {
		t.Skipf("tor not available: %v", err)
	}
	t.Logf("Detected Tor version: %s", v)

	if v.Major == 0 && v.Minor == 0 {
		t.Fatal("version should not be 0.0.x")
	}
}

func TestIntegration_StartStopInstance(t *testing.T) {
	cfg := config.Defaults()
	cfg.Paths.TempFiles = t.TempDir()
	cfg.Tor.BinaryPath = "/usr/bin/tor"

	procMgr := process.NewManager(cfg.Paths.TempFiles)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	v, err := DetectVersion(ctx, cfg.Tor.BinaryPath)
	if err != nil {
		t.Skipf("tor not available: %v", err)
	}

	mgr := NewManager(cfg, procMgr)
	mgr.CreateFromVersion(v, []string{"{US}"})

	instances := mgr.GetInstances()
	if len(instances) == 0 {
		t.Fatal("expected at least one instance")
	}

	inst := instances[0]
	if err := inst.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	time.Sleep(2 * time.Second)

	stopCtx, stopCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer stopCancel()
	if err := inst.Stop(stopCtx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
