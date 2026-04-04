package tor

import (
	"testing"

	"github.com/user/splitter/internal/process"
)

func TestManager_CreateFromVersion(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	countries := []string{"{US}", "{DE}", "{FR}"}

	mgr.CreateFromVersion(v, countries)

	instances := mgr.GetInstances()
	expectedCount := 3 * 2 // 3 countries * 2 per country
	if len(instances) != expectedCount {
		t.Fatalf("GetInstances() returned %d, want %d", len(instances), expectedCount)
	}

	for i, inst := range instances {
		expectedSocks := 4999 + i
		expectedControl := 5999 + i
		expectedHTTP := 5199 + i

		if inst.SocksPort != expectedSocks {
			t.Errorf("instance[%d].SocksPort = %d, want %d", i, inst.SocksPort, expectedSocks)
		}
		if inst.ControlPort != expectedControl {
			t.Errorf("instance[%d].ControlPort = %d, want %d", i, inst.ControlPort, expectedControl)
		}
		if inst.HTTPPort != expectedHTTP {
			t.Errorf("instance[%d].HTTPPort = %d, want %d", i, inst.HTTPPort, expectedHTTP)
		}
	}
}

func TestManager_CreateFromVersion_OldTor(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 7, Release: 0}
	countries := []string{"{US}"}

	mgr.CreateFromVersion(v, countries)

	instances := mgr.GetInstances()
	if len(instances) != 1 {
		t.Fatalf("GetInstances() returned %d, want 1", len(instances))
	}

	inst := instances[0]
	if inst.HTTPPort != 0 {
		t.Errorf("HTTPPort = %d, want 0 for Tor 0.4.7 (no HTTPTunnelPort)", inst.HTTPPort)
	}
}

func TestManager_GetInstance(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	countries := []string{"{US}", "{DE}"}

	mgr.CreateFromVersion(v, countries)

	inst, err := mgr.GetInstance(2)
	if err != nil {
		t.Fatalf("GetInstance(2) error = %v", err)
	}
	if inst.Country != "{DE}" {
		t.Errorf("GetInstance(2).Country = %q, want %q", inst.Country, "{DE}")
	}
}

func TestManager_GetInstance_NotFound(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	mgr.CreateFromVersion(&Version{0, 4, 8, 0}, []string{"{US}"})

	_, err := mgr.GetInstance(999)
	if err == nil {
		t.Error("GetInstance(999) expected error, got nil")
	}
}

func TestManager_GetVersion(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 9, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}"})

	got := mgr.GetVersion()
	if got.String() != "0.4.9.0" {
		t.Errorf("GetVersion() = %v, want 0.4.9.0", got)
	}
}

func TestManager_PortIncrement(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 3
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	countries := []string{"{US}", "{DE}"}

	mgr.CreateFromVersion(v, countries)

	instances := mgr.GetInstances()
	if len(instances) != 6 {
		t.Fatalf("expected 6 instances, got %d", len(instances))
	}

	last := instances[5]
	if last.SocksPort != 4999+5 {
		t.Errorf("last instance SocksPort = %d, want %d", last.SocksPort, 4999+5)
	}
	if last.ControlPort != 5999+5 {
		t.Errorf("last instance ControlPort = %d, want %d", last.ControlPort, 5999+5)
	}
}

func TestManager_CountryAssignment(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	countries := []string{"{US}", "{DE}", "{FR}"}

	mgr.CreateFromVersion(v, countries)

	instances := mgr.GetInstances()

	expectedCountries := []string{"{US}", "{US}", "{DE}", "{DE}", "{FR}", "{FR}"}
	for i, inst := range instances {
		if inst.Country != expectedCountries[i] {
			t.Errorf("instance[%d].Country = %q, want %q", i, inst.Country, expectedCountries[i])
		}
	}
}

func TestManager_EmptyCountries(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	mgr.CreateFromVersion(&Version{0, 4, 8, 0}, []string{})

	instances := mgr.GetInstances()
	if len(instances) != 0 {
		t.Errorf("expected 0 instances for empty countries, got %d", len(instances))
	}
}
