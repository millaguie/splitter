package tor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/user/splitter/internal/process"
)

func writeBridgesYAML(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "bridges.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write bridges yaml: %v", err)
	}
	return path
}

const validBridgesYAML = `
snowflake:
  description: "Snowflake WebRTC transport"
  transport: "snowflake"
  lines:
    - "Bridge snowflake 192.0.2.1:80 192.0.2.1:443 fingerprint=fingerprint1"
    - "Bridge snowflake 192.0.2.2:80 192.0.2.2:443 fingerprint=fingerprint2"

webtunnel:
  description: "WebTunnel HTTPS transport"
  transport: "webtunnel"
  lines:
    - "Bridge webtunnel 192.0.2.3:443 192.0.2.3:443 fingerprint=fingerprint3 url=https://example.com/tor"

obfs4:
  description: "obfs4 transport"
  transport: "obfs4"
  lines:
    - "Bridge obfs4 192.0.2.4:443 192.0.2.4:443 fingerprint=fingerprint4 cert=cert1 iat-mode=0"
    - "Bridge obfs4 192.0.2.5:443 192.0.2.5:443 fingerprint=fingerprint5 cert=cert2 iat-mode=0"
`

func TestLoadBridges_ValidFile(t *testing.T) {
	path := writeBridgesYAML(t, validBridgesYAML)
	cfg, err := LoadBridges(path)
	if err != nil {
		t.Fatalf("LoadBridges() error = %v", err)
	}
	if cfg.Snowflake == nil {
		t.Error("Snowflake is nil")
	}
	if cfg.WebTunnel == nil {
		t.Error("WebTunnel is nil")
	}
	if cfg.Obfs4 == nil {
		t.Error("Obfs4 is nil")
	}
}

func TestLoadBridges_MissingFile(t *testing.T) {
	_, err := LoadBridges("/nonexistent/bridges.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestLoadBridges_InvalidYAML(t *testing.T) {
	path := writeBridgesYAML(t, "not: [valid: yaml {{{")
	_, err := LoadBridges(path)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestBridgesConfig_GetBridge_Snowflake(t *testing.T) {
	path := writeBridgesYAML(t, validBridgesYAML)
	cfg, _ := LoadBridges(path)

	bc, err := cfg.GetBridge("snowflake")
	if err != nil {
		t.Fatalf("GetBridge(snowflake) error = %v", err)
	}
	if bc.Transport != "snowflake" {
		t.Errorf("Transport = %q, want %q", bc.Transport, "snowflake")
	}
	if len(bc.Lines) != 2 {
		t.Errorf("len(Lines) = %d, want 2", len(bc.Lines))
	}
}

func TestBridgesConfig_GetBridge_WebTunnel(t *testing.T) {
	path := writeBridgesYAML(t, validBridgesYAML)
	cfg, _ := LoadBridges(path)

	bc, err := cfg.GetBridge("webtunnel")
	if err != nil {
		t.Fatalf("GetBridge(webtunnel) error = %v", err)
	}
	if bc.Transport != "webtunnel" {
		t.Errorf("Transport = %q, want %q", bc.Transport, "webtunnel")
	}
	if len(bc.Lines) != 1 {
		t.Errorf("len(Lines) = %d, want 1", len(bc.Lines))
	}
}

func TestBridgesConfig_GetBridge_Obfs4(t *testing.T) {
	path := writeBridgesYAML(t, validBridgesYAML)
	cfg, _ := LoadBridges(path)

	bc, err := cfg.GetBridge("obfs4")
	if err != nil {
		t.Fatalf("GetBridge(obfs4) error = %v", err)
	}
	if bc.Transport != "obfs4" {
		t.Errorf("Transport = %q, want %q", bc.Transport, "obfs4")
	}
	if len(bc.Lines) != 2 {
		t.Errorf("len(Lines) = %d, want 2", len(bc.Lines))
	}
}

func TestBridgesConfig_GetBridge_None(t *testing.T) {
	cfg := &BridgesConfig{}

	bc, err := cfg.GetBridge("none")
	if err != nil {
		t.Fatalf("GetBridge(none) error = %v", err)
	}
	if bc != nil {
		t.Error("expected nil for none bridge type")
	}
}

func TestBridgesConfig_GetBridge_Empty(t *testing.T) {
	cfg := &BridgesConfig{}

	bc, err := cfg.GetBridge("")
	if err != nil {
		t.Fatalf("GetBridge('') error = %v", err)
	}
	if bc != nil {
		t.Error("expected nil for empty bridge type")
	}
}

func TestBridgesConfig_GetBridge_Unknown(t *testing.T) {
	cfg := &BridgesConfig{}

	_, err := cfg.GetBridge("unknown")
	if err == nil {
		t.Error("expected error for unknown bridge type, got nil")
	}
	if !strings.Contains(err.Error(), "unknown bridge type") {
		t.Errorf("error = %q, want to contain 'unknown bridge type'", err.Error())
	}
}

func TestTorrcTemplate_WithBridges(t *testing.T) {
	ic := baseInstanceConfig()
	ic.UseBridges = true
	ic.BridgeLines = []string{
		"Bridge snowflake 192.0.2.1:80 fingerprint1",
		"Bridge snowflake 192.0.2.2:80 fingerprint2",
	}
	ic.ClientTransport = "snowflake"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "UseBridges 1")
	torrcContains(t, result, "Bridge snowflake 192.0.2.1:80 fingerprint1")
	torrcContains(t, result, "Bridge snowflake 192.0.2.2:80 fingerprint2")
	torrcContains(t, result, "ClientTransportPlugin snowflake exec /usr/bin/lyrebird")
}

func TestTorrcTemplate_NoBridges(t *testing.T) {
	ic := baseInstanceConfig()
	ic.UseBridges = false
	ic.BridgeLines = nil
	ic.ClientTransport = ""

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "UseBridges")
	torrcNotContains(t, result, "ClientTransportPlugin")
}

func TestTorrcTemplate_WithObfs4Bridges(t *testing.T) {
	ic := baseInstanceConfig()
	ic.UseBridges = true
	ic.BridgeLines = []string{
		"Bridge obfs4 192.0.2.4:443 fingerprint4 cert=cert1 iat-mode=0",
	}
	ic.ClientTransport = "obfs4"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "UseBridges 1")
	torrcContains(t, result, "Bridge obfs4 192.0.2.4:443")
	torrcContains(t, result, "ClientTransportPlugin obfs4 exec /usr/bin/lyrebird")
}

func TestInstance_SetBridges_Snowflake(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	path := writeBridgesYAML(t, validBridgesYAML)
	bridges, _ := LoadBridges(path)

	err := inst.SetBridges("snowflake", bridges)
	if err != nil {
		t.Fatalf("SetBridges(snowflake) error = %v", err)
	}
	if len(inst.bridgeLines) != 2 {
		t.Errorf("bridgeLines len = %d, want 2", len(inst.bridgeLines))
	}
	if inst.bridgeTransport != "snowflake" {
		t.Errorf("bridgeTransport = %q, want %q", inst.bridgeTransport, "snowflake")
	}
}

func TestInstance_SetBridges_None(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	err := inst.SetBridges("none", nil)
	if err != nil {
		t.Fatalf("SetBridges(none) error = %v", err)
	}
	if len(inst.bridgeLines) != 0 {
		t.Errorf("bridgeLines len = %d, want 0", len(inst.bridgeLines))
	}
}

func TestInstance_SetBridges_Empty(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	err := inst.SetBridges("", nil)
	if err != nil {
		t.Fatalf("SetBridges('') error = %v", err)
	}
	if len(inst.bridgeLines) != 0 {
		t.Errorf("bridgeLines len = %d, want 0", len(inst.bridgeLines))
	}
}

func TestInstance_SetBridges_NilConfig(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	err := inst.SetBridges("snowflake", nil)
	if err == nil {
		t.Error("expected error for nil bridges config, got nil")
	}
}

func TestInstance_SetBridges_Unknown(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	path := writeBridgesYAML(t, validBridgesYAML)
	bridges, _ := LoadBridges(path)

	err := inst.SetBridges("unknown", bridges)
	if err == nil {
		t.Error("expected error for unknown bridge type, got nil")
	}
}

func TestBuildInstanceConfig_WithBridges(t *testing.T) {
	cfg := testConfig(t)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	inst.SocksPort = 4999
	inst.ControlPort = 5999
	inst.bridgeLines = []string{"Bridge snowflake 192.0.2.1:80 fp1"}
	inst.bridgeTransport = "snowflake"

	ic := inst.buildInstanceConfig()

	if !ic.UseBridges {
		t.Error("UseBridges = false, want true")
	}
	if len(ic.BridgeLines) != 1 {
		t.Errorf("BridgeLines len = %d, want 1", len(ic.BridgeLines))
	}
	if ic.ClientTransport != "snowflake" {
		t.Errorf("ClientTransport = %q, want %q", ic.ClientTransport, "snowflake")
	}
}

func TestBuildInstanceConfig_NoBridges(t *testing.T) {
	cfg := testConfig(t)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	inst := NewInstance(0, "{US}", cfg, v, process.NewManager(""))
	inst.SocksPort = 4999
	inst.ControlPort = 5999

	ic := inst.buildInstanceConfig()

	if ic.UseBridges {
		t.Error("UseBridges = true, want false")
	}
	if len(ic.BridgeLines) != 0 {
		t.Errorf("BridgeLines len = %d, want 0", len(ic.BridgeLines))
	}
}

func TestManager_SetBridgesForAll(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}", "{DE}"})

	path := writeBridgesYAML(t, validBridgesYAML)
	bridges, _ := LoadBridges(path)

	err := mgr.SetBridgesForAll("obfs4", bridges)
	if err != nil {
		t.Fatalf("SetBridgesForAll(obfs4) error = %v", err)
	}

	for _, inst := range mgr.GetInstances() {
		if len(inst.bridgeLines) != 2 {
			t.Errorf("instance %d bridgeLines len = %d, want 2", inst.ID, len(inst.bridgeLines))
		}
		if inst.bridgeTransport != "obfs4" {
			t.Errorf("instance %d bridgeTransport = %q, want %q", inst.ID, inst.bridgeTransport, "obfs4")
		}
	}
}

func TestManager_SetBridgesForAll_None(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)
	mgr.CreateFromVersion(&Version{0, 4, 8, 0}, []string{"{US}"})

	err := mgr.SetBridgesForAll("none", nil)
	if err != nil {
		t.Fatalf("SetBridgesForAll(none) error = %v", err)
	}

	for _, inst := range mgr.GetInstances() {
		if len(inst.bridgeLines) != 0 {
			t.Errorf("instance %d bridgeLines len = %d, want 0", inst.ID, len(inst.bridgeLines))
		}
	}
}
