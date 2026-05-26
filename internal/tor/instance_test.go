package tor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Tor.BinaryPath = "/usr/bin/tor"
	cfg.Tor.ListenAddr = "0.0.0.0"
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199
	cfg.Tor.ControlAuth = "cookie"
	cfg.Tor.HiddenService.Enabled = true
	cfg.Tor.HiddenService.BasePath = "/tmp/splitter/hidden_service_"
	cfg.Tor.HiddenService.StartPort = 3999
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
	cfg.Tor.AutomapHostsSuffixes = ".exit,.onion"
	cfg.Tor.WarnPlaintextPorts = "21,23,25,80,109,110,143"
	cfg.Tor.RejectPlaintextPorts = ""
	cfg.Tor.ConfluxEnabled = true
	cfg.Tor.CongestionControlAuto = true
	cfg.Relay.Enforce = "entry"
	cfg.Paths.TempFiles = t.TempDir()
	return cfg
}

func TestInstance_PortAssignment(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(3, "{US}", cfg, v, procMgr)
	inst.SocksPort = cfg.Tor.StartSocksPort + 3
	inst.ControlPort = cfg.Tor.StartControlPort + 3
	inst.HTTPPort = cfg.Tor.StartHTTPPort + 3

	if inst.SocksPort != 5002 {
		t.Errorf("SocksPort = %d, want 5002", inst.SocksPort)
	}
	if inst.ControlPort != 6002 {
		t.Errorf("ControlPort = %d, want 6002", inst.ControlPort)
	}
	if inst.HTTPPort != 5202 {
		t.Errorf("HTTPPort = %d, want 5202", inst.HTTPPort)
	}
}

func TestInstance_StateTransitions(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	if inst.GetState() != StateStarting {
		t.Errorf("initial state = %v, want StateStarting", inst.GetState())
	}

	inst.setState(StateBootstrapping)
	if inst.GetState() != StateBootstrapping {
		t.Errorf("state = %v, want StateBootstrapping", inst.GetState())
	}

	inst.setState(StateReady)
	if inst.GetState() != StateReady {
		t.Errorf("state = %v, want StateReady", inst.GetState())
	}

	inst.setState(StateFailed)
	if inst.GetState() != StateFailed {
		t.Errorf("state = %v, want StateFailed", inst.GetState())
	}
}

func TestInstance_GenerateDataDir(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(5, "{DE}", cfg, v, procMgr)

	dataDir, err := inst.generateDataDir()
	if err != nil {
		t.Fatalf("generateDataDir() error = %v", err)
	}

	expected := filepath.Join(cfg.Paths.TempFiles, "tor_data_5")
	if dataDir != expected {
		t.Errorf("dataDir = %q, want %q", dataDir, expected)
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		t.Errorf("data dir %q was not created", dataDir)
	}
}

func TestBuildInstanceConfig_EntryMode(t *testing.T) {
	cfg := testConfig(t)
	cfg.Relay.Enforce = "entry"
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	inst.SocksPort = 4999
	inst.ControlPort = 5999
	inst.HTTPPort = 5199

	ic := inst.buildInstanceConfig()

	if ic.RelayEnforce != "entry" {
		t.Errorf("RelayEnforce = %q, want %q", ic.RelayEnforce, "entry")
	}
	if ic.Country != "{US}" {
		t.Errorf("Country = %q, want %q", ic.Country, "{US}")
	}
	if !ic.CongestionControlAuto {
		t.Error("CongestionControlAuto = false, want true for 0.4.8")
	}
	if !ic.ConfluxEnabled {
		t.Error("ConfluxEnabled = false, want true for 0.4.8")
	}
	if !ic.HiddenServiceEnabled {
		t.Error("HiddenServiceEnabled = false, want true")
	}
	if ic.HTTPTunnelPort != 5199 {
		t.Errorf("HTTPTunnelPort = %d, want 5199", ic.HTTPTunnelPort)
	}
}

func TestBuildInstanceConfig_ExitMode(t *testing.T) {
	cfg := testConfig(t)
	cfg.Relay.Enforce = "exit"
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(1, "{DE}", cfg, v, process.NewManager(""))
	inst.SocksPort = 5000
	inst.ControlPort = 6000
	inst.HTTPPort = 5200

	ic := inst.buildInstanceConfig()

	if ic.RelayEnforce != "exit" {
		t.Errorf("RelayEnforce = %q, want %q", ic.RelayEnforce, "exit")
	}
}

func TestBuildInstanceConfig_SpeedMode(t *testing.T) {
	cfg := testConfig(t)
	cfg.Relay.Enforce = "speed"
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(2, "{FR}", cfg, v, process.NewManager(""))
	ic := inst.buildInstanceConfig()

	if ic.RelayEnforce != "speed" {
		t.Errorf("RelayEnforce = %q, want %q", ic.RelayEnforce, "speed")
	}
}

func TestBuildInstanceConfig_OldTorNoHTTPTunnel(t *testing.T) {
	cfg := testConfig(t)
	v := &Version{Major: 0, Minor: 4, Patch: 7, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	inst.SocksPort = 4999
	inst.ControlPort = 5999
	inst.HTTPPort = 0

	ic := inst.buildInstanceConfig()

	if ic.HTTPTunnelPort != 0 {
		t.Errorf("HTTPTunnelPort = %d, want 0 for Tor 0.4.7", ic.HTTPTunnelPort)
	}
	if !ic.CongestionControlAuto {
		t.Error("CongestionControlAuto = false, want true for 0.4.7")
	}
	if ic.ConfluxEnabled {
		t.Error("ConfluxEnabled = true, want false for 0.4.7")
	}
}

func TestBuildInstanceConfig_CGORRequires049(t *testing.T) {
	cfg := testConfig(t)
	v := &Version{Major: 0, Minor: 4, Patch: 9, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	ic := inst.buildInstanceConfig()

	if !ic.CongestionControlAuto {
		t.Error("CongestionControlAuto should be true for 0.4.9")
	}
	if !ic.ConfluxEnabled {
		t.Error("ConfluxEnabled should be true for 0.4.9")
	}
}

func TestBuildInstanceConfig_HiddenServiceDisabled(t *testing.T) {
	cfg := testConfig(t)
	cfg.Tor.HiddenService.Enabled = false
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	ic := inst.buildInstanceConfig()

	if ic.HiddenServiceEnabled {
		t.Error("HiddenServiceEnabled should be false when disabled in config")
	}
}

func TestRenderTorrc_EntryMode(t *testing.T) {
	ic := InstanceConfig{
		InstanceID:                    0,
		Country:                       "{US}",
		SocksPort:                     4999,
		ControlPort:                   5999,
		HTTPTunnelPort:                5199,
		DataDir:                       "/tmp/splitter/tor_data_0",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           15,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         true,
		ConfluxEnabled:                true,
		RelayEnforce:                  "entry",
		HiddenServiceEnabled:          true,
		HiddenServiceDir:              "/tmp/splitter/hidden_service_0",
		HiddenServicePort:             3999,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               0,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25,80,109,110,143",
		RejectPlaintextPorts:          "",
		KeepalivePeriod:               15,
		ControlAuth:                   "cookie",
	}

	tmpl := readFile(t, "../../templates/torrc.gotmpl")
	result, err := RenderTorrc(ic, tmpl)
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	assertContains(t, result, "SocksPort 4999")
	assertContains(t, result, "ControlPort 5999")
	assertContains(t, result, "HTTPTunnelPort 5199")
	assertContains(t, result, "EntryNodes {US}")
	assertContains(t, result, "CongestionControlAuto 1")
	assertContains(t, result, "ConfluxEnabled 1")
	assertContains(t, result, "CookieAuthentication 1")
	assertContains(t, result, "HiddenServiceDir /tmp/splitter/hidden_service_0")
	assertContains(t, result, "HiddenServicePort 3999")

	assertNotContains(t, result, "ExitNodes")
	assertNotContains(t, result, "RejectPlaintextPorts")
}

func TestRenderTorrc_ExitMode(t *testing.T) {
	ic := InstanceConfig{
		InstanceID:                    1,
		Country:                       "{DE}",
		SocksPort:                     5000,
		ControlPort:                   6000,
		HTTPTunnelPort:                0,
		DataDir:                       "/tmp/splitter/tor_data_1",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           15,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         true,
		ConfluxEnabled:                true,
		RelayEnforce:                  "exit",
		HiddenServiceEnabled:          false,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               0,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25,80,109,110,143",
		RejectPlaintextPorts:          "",
		KeepalivePeriod:               15,
	}

	tmpl := readFile(t, "../../templates/torrc.gotmpl")
	result, err := RenderTorrc(ic, tmpl)
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	assertContains(t, result, "ExitNodes {DE}")
	assertNotContains(t, result, "EntryNodes")
	assertNotContains(t, result, "HTTPTunnelPort")
	assertNotContains(t, result, "HiddenServiceDir")
}

func TestRenderTorrc_SpeedMode(t *testing.T) {
	ic := InstanceConfig{
		InstanceID:                    2,
		Country:                       "{FR}",
		SocksPort:                     5001,
		ControlPort:                   6001,
		HTTPTunnelPort:                0,
		DataDir:                       "/tmp/splitter/tor_data_2",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           15,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         false,
		ConfluxEnabled:                false,
		RelayEnforce:                  "speed",
		HiddenServiceEnabled:          false,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               0,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25,80,109,110,143",
		RejectPlaintextPorts:          "",
		KeepalivePeriod:               15,
	}

	tmpl := readFile(t, "../../templates/torrc.gotmpl")
	result, err := RenderTorrc(ic, tmpl)
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	assertContains(t, result, "speed mode")
	assertNotContains(t, result, "EntryNodes")
	assertNotContains(t, result, "ExitNodes")
	assertNotContains(t, result, "CongestionControlAuto")
	assertNotContains(t, result, "ConfluxEnabled")
}

func TestRenderTorrc_WithRejectPlaintextPorts(t *testing.T) {
	ic := InstanceConfig{
		InstanceID:                    0,
		Country:                       "{US}",
		SocksPort:                     4999,
		ControlPort:                   5999,
		HTTPTunnelPort:                0,
		DataDir:                       "/tmp/splitter/tor_data_0",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           15,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         false,
		ConfluxEnabled:                false,
		RelayEnforce:                  "entry",
		HiddenServiceEnabled:          false,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               0,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25,80,109,110,143",
		RejectPlaintextPorts:          "21,23,25",
		KeepalivePeriod:               15,
	}

	tmpl := readFile(t, "../../templates/torrc.gotmpl")
	result, err := RenderTorrc(ic, tmpl)
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	assertContains(t, result, "RejectPlaintextPorts 21,23,25")
}

func TestRenderTorrc_FascistFirewall(t *testing.T) {
	ic := InstanceConfig{
		InstanceID:                    0,
		Country:                       "{US}",
		SocksPort:                     4999,
		ControlPort:                   5999,
		HTTPTunnelPort:                0,
		DataDir:                       "/tmp/splitter/tor_data_0",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           15,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         false,
		ConfluxEnabled:                false,
		RelayEnforce:                  "entry",
		HiddenServiceEnabled:          false,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               1,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25,80,109,110,143",
		RejectPlaintextPorts:          "",
		KeepalivePeriod:               15,
	}

	tmpl := readFile(t, "../../templates/torrc.gotmpl")
	result, err := RenderTorrc(ic, tmpl)
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	assertContains(t, result, "FascistFirewall 1")
	assertContains(t, result, "FirewallPorts 80,443")
}

func TestRandomizeDirtiness(t *testing.T) {
	cfg := testConfig(t)
	cfg.Tor.MaxCircuitDirtiness = 30
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))

	for i := 0; i < 100; i++ {
		d := inst.randomizeDirtiness()
		if d < 10 || d > 30 {
			t.Errorf("randomizeDirtiness() = %d, want range [10, 30]", d)
		}
	}
}

func TestRandomizeDirtiness_SmallMax(t *testing.T) {
	cfg := testConfig(t)
	cfg.Tor.MaxCircuitDirtiness = 5
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))

	d := inst.randomizeDirtiness()
	if d != 5 {
		t.Errorf("randomizeDirtiness() = %d, want 5 when max <= 10", d)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("expected output NOT to contain %q\nfull output:\n%s", needle, haystack)
	}
}
