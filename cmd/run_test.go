package cmd

import (
	"context"
	"testing"
	"time"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/country"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

func TestNewRunCmd_Structure(t *testing.T) {
	cmd := newRunCmd()

	if cmd.Use != "run" {
		t.Errorf("Use = %q, want %q", cmd.Use, "run")
	}
	if cmd.Short == "" {
		t.Error("Short should not be empty")
	}
	if cmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	ci, err := cmd.Flags().GetDuration("country-interval")
	if err != nil {
		t.Fatalf("country-interval flag error: %v", err)
	}
	if ci != 120*time.Second {
		t.Errorf("country-interval = %v, want %v", ci, 120*time.Second)
	}

	lb, err := cmd.Flags().GetString("load-balance")
	if err != nil {
		t.Fatalf("load-balance flag error: %v", err)
	}
	if lb != "roundrobin" {
		t.Errorf("load-balance = %q, want %q", lb, "roundrobin")
	}
}

func TestNewRunCmd_FlagsChanged(t *testing.T) {
	cmd := newRunCmd()
	cmd.SetArgs([]string{"--country-interval", "60s", "--load-balance", "leastconn"})

	if err := cmd.ParseFlags([]string{"--country-interval", "60s", "--load-balance", "leastconn"}); err != nil {
		t.Fatalf("ParseFlags error: %v", err)
	}

	if !cmd.Flags().Changed("country-interval") {
		t.Error("country-interval flag should be changed")
	}
	if !cmd.Flags().Changed("load-balance") {
		t.Error("load-balance flag should be changed")
	}
}

func TestTorRotator_GetInstances(t *testing.T) {
	cfg := testRunConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	tm := tor.NewManager(cfg, procMgr)
	tm.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 8}, []string{"{US}", "{DE}"})

	r := &torRotator{tm: tm}
	instances := r.GetInstances()

	if len(instances) != 2 {
		t.Fatalf("GetInstances() returned %d, want 2", len(instances))
	}

	if instances[0].ID != 0 || instances[0].Country != "{US}" {
		t.Errorf("instances[0] = {ID: %d, Country: %q}, want {0, \"{US}\"}", instances[0].ID, instances[0].Country)
	}
	if instances[1].ID != 1 || instances[1].Country != "{DE}" {
		t.Errorf("instances[1] = {ID: %d, Country: %q}, want {1, \"{DE}\"}", instances[1].ID, instances[1].Country)
	}
}

func TestTorRotator_GetInstances_Empty(t *testing.T) {
	cfg := testRunConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	tm := tor.NewManager(cfg, procMgr)

	r := &torRotator{tm: tm}
	instances := r.GetInstances()

	if len(instances) != 0 {
		t.Errorf("GetInstances() returned %d, want 0", len(instances))
	}
}

func TestTorRotator_RotateInstance_NotFound(t *testing.T) {
	cfg := testRunConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	tm := tor.NewManager(cfg, procMgr)

	r := &torRotator{tm: tm}
	err := r.RotateInstance(context.Background(), 999, "{FR}")
	if err == nil {
		t.Error("RotateInstance(999) expected error, got nil")
	}
}

func TestTorRotator_RotateInstance_UpdatesCountry(t *testing.T) {
	cfg := testRunConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	tm := tor.NewManager(cfg, procMgr)
	tm.CreateFromVersion(&tor.Version{Major: 0, Minor: 4, Patch: 8}, []string{"{US}"})

	r := &torRotator{tm: tm}
	_ = r.RotateInstance(context.Background(), 0, "{GB}")

	instances := r.GetInstances()
	if len(instances) != 1 {
		t.Fatalf("GetInstances() returned %d, want 1", len(instances))
	}
	if instances[0].Country != "{GB}" {
		t.Errorf("Country = %q, want %q", instances[0].Country, "{GB}")
	}
}

func TestTorRotator_ImplementsInterface(t *testing.T) {
	cfg := testRunConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	tm := tor.NewManager(cfg, procMgr)

	var _ country.InstanceRotator = &torRotator{tm: tm}
}

func testRunConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Tor.BinaryPath = "/usr/bin/tor"
	cfg.Tor.ListenAddr = "0.0.0.0"
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199
	cfg.Tor.ControlAuth = "cookie"
	cfg.Tor.HiddenService.Enabled = false
	cfg.Tor.MinimumTimeout = 15
	cfg.Tor.CircuitBuildTimeout = 60
	cfg.Tor.CircuitStreamTimeout = 20
	cfg.Tor.MaxCircuitDirtiness = 30
	cfg.Tor.NewCircuitPeriod = 30
	cfg.Tor.LearnCircuitBuildTimeout = 1
	cfg.Tor.ClientOnly = 0
	cfg.Tor.ConnectionPadding = 0
	cfg.Tor.ReducedConnectionPadding = 1
	cfg.Tor.GeoIPExcludeUnknown = 1
	cfg.Tor.StrictNodes = 1
	cfg.Tor.FascistFirewall = 0
	cfg.Tor.FirewallPorts = []int{80, 443}
	cfg.Tor.LongLivedPorts = []int{1, 2}
	cfg.Tor.MaxClientCircuitsPending = 1024
	cfg.Tor.SocksTimeout = 35
	cfg.Tor.TrackHostExitsExpire = 10
	cfg.Tor.UseEntryGuards = 1
	cfg.Tor.NumEntryGuards = 1
	cfg.Tor.SafeSocks = 1
	cfg.Tor.TestSocks = 1
	cfg.Tor.ClientRejectInternalAddresses = 1
	cfg.Tor.OptimisticData = "auto"
	cfg.Tor.AutomapHostsSuffixes = ".exit,.onion"
	cfg.Tor.WarnPlaintextPorts = "21,23,25,80,109,110,143"
	cfg.Tor.RejectPlaintextPorts = ""
	cfg.Relay.Enforce = "entry"
	cfg.Paths.TempFiles = t.TempDir()
	cfg.Instances.PerCountry = 1
	return cfg
}
