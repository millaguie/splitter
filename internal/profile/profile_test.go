package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConstants(t *testing.T) {
	if Stealth != "stealth" {
		t.Errorf("Stealth = %q, want %q", Stealth, "stealth")
	}
	if Balanced != "balanced" {
		t.Errorf("Balanced = %q, want %q", Balanced, "balanced")
	}
	if Streaming != "streaming" {
		t.Errorf("Streaming = %q, want %q", Streaming, "streaming")
	}
	if Pentest != "pentest" {
		t.Errorf("Pentest = %q, want %q", Pentest, "pentest")
	}
}

func TestNames(t *testing.T) {
	names := Names()
	if len(names) != 4 {
		t.Fatalf("Names() returned %d names, want 4", len(names))
	}
	expected := []string{"stealth", "balanced", "streaming", "pentest"}
	for i, e := range expected {
		if names[i] != e {
			t.Errorf("Names()[%d] = %q, want %q", i, names[i], e)
		}
	}
}

func TestNames_ReturnsCopy(t *testing.T) {
	n1 := Names()
	n1[0] = "mutated"
	n2 := Names()
	if n2[0] == "mutated" {
		t.Error("Names() should return a copy, not the original slice")
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"stealth", true},
		{"balanced", true},
		{"streaming", true},
		{"pentest", true},
		{"", false},
		{"unknown", false},
		{"Stealth", false},
		{"BALANCED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValid(tt.name); got != tt.want {
				t.Errorf("IsValid(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestValidProfiles(t *testing.T) {
	if len(ValidProfiles) != 4 {
		t.Errorf("ValidProfiles has %d entries, want 4", len(ValidProfiles))
	}
}

func TestLoad_FromActualProfilesFile(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, name := range ValidProfiles {
		p, ok := profiles[name]
		if !ok {
			t.Errorf("profile %q not found in profiles.yaml", name)
			continue
		}
		if p.Description == "" {
			t.Errorf("profile %q has empty description", name)
		}
	}
}

func TestLoad_StealthProfileFields(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	s := profiles["stealth"]
	if s == nil {
		t.Fatal("stealth profile is nil")
	}

	if s.Instances.PerCountry == nil || *s.Instances.PerCountry != 3 {
		t.Error("stealth instances.per_country should be 3")
	}
	if s.Instances.Countries == nil || *s.Instances.Countries != 8 {
		t.Error("stealth instances.countries should be 8")
	}
	if s.Tor.ConfluxEnabled == nil || !*s.Tor.ConfluxEnabled {
		t.Error("stealth tor.conflux_enabled should be true")
	}
	if s.Tor.CongestionControlAuto == nil || !*s.Tor.CongestionControlAuto {
		t.Error("stealth tor.congestion_control_auto should be true")
	}
	if s.Tor.Sandbox == nil || !*s.Tor.Sandbox {
		t.Error("stealth tor.sandbox should be true")
	}
	if s.Tor.CircuitFingerprintingResistance == nil || !*s.Tor.CircuitFingerprintingResistance {
		t.Error("stealth tor.circuit_fingerprinting_resistance should be true")
	}
	if s.Country.RotationInterval == nil || *s.Country.RotationInterval != 60 {
		t.Error("stealth country.rotation_interval should be 60")
	}
	if s.Country.TotalToChange == nil || *s.Country.TotalToChange != 5 {
		t.Error("stealth country.total_to_change should be 5")
	}
}

func TestLoad_BalancedProfileFields(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	b := profiles["balanced"]
	if b == nil {
		t.Fatal("balanced profile is nil")
	}
	if b.Tor.ConfluxEnabled == nil || *b.Tor.ConfluxEnabled {
		t.Error("balanced tor.conflux_enabled should be false")
	}
	if b.Tor.CongestionControlAuto == nil || !*b.Tor.CongestionControlAuto {
		t.Error("balanced tor.congestion_control_auto should be true")
	}
	if b.Tor.Sandbox == nil || *b.Tor.Sandbox {
		t.Error("balanced tor.sandbox should be false")
	}
}

func TestLoad_StreamingProfileFields(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	s := profiles["streaming"]
	if s == nil {
		t.Fatal("streaming profile is nil")
	}
	if s.Tor.ConfluxEnabled == nil || !*s.Tor.ConfluxEnabled {
		t.Error("streaming tor.conflux_enabled should be true")
	}
	if s.Tor.CongestionControlAuto == nil || !*s.Tor.CongestionControlAuto {
		t.Error("streaming tor.congestion_control_auto should be true")
	}
	if s.Tor.IPv6 == nil || !*s.Tor.IPv6 {
		t.Error("streaming tor.ipv6 should be true")
	}
	if s.Tor.Sandbox == nil || *s.Tor.Sandbox {
		t.Error("streaming tor.sandbox should be false")
	}
}

func TestLoad_PentestProfileFields(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	p := profiles["pentest"]
	if p == nil {
		t.Fatal("pentest profile is nil")
	}
	if p.Tor.ConfluxEnabled == nil || *p.Tor.ConfluxEnabled {
		t.Error("pentest tor.conflux_enabled should be false")
	}
	if p.Tor.CongestionControlAuto == nil || *p.Tor.CongestionControlAuto {
		t.Error("pentest tor.congestion_control_auto should be false")
	}
	if p.Tor.StreamIsolation == nil || !*p.Tor.StreamIsolation {
		t.Error("pentest tor.stream_isolation should be true")
	}
	if p.Tor.CircuitFingerprintingResistance == nil || !*p.Tor.CircuitFingerprintingResistance {
		t.Error("pentest tor.circuit_fingerprinting_resistance should be true")
	}
	if p.HealthCheck.ExitReputation == nil || !*p.HealthCheck.ExitReputation {
		t.Error("pentest health_check.exit_reputation should be true")
	}
	if p.Country.RotationInterval == nil || *p.Country.RotationInterval != 60 {
		t.Error("pentest country.rotation_interval should be 60")
	}
}

func TestProfileStruct_FieldParsing(t *testing.T) {
	yaml := `
testprof:
  description: "test profile"
  instances:
    per_country: 7
    countries: 3
  relay:
    enforce: "exit"
  proxy:
    load_balance_algorithm: "leastconn"
  tor:
    max_circuit_dirtiness: 42
    connection_padding: 1
    use_entry_guards: 0
    reduced_connection_padding: 0
    stream_isolation: true
    ipv6: true
    conflux_enabled: true
    congestion_control_auto: true
    sandbox: true
    circuit_fingerprinting_resistance: true
  logging:
    enabled: true
    level: "DEBUG"
  country:
    rotation_interval: 90
    total_to_change: 3
  health_check:
    exit_reputation: true
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	profiles, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	p, ok := profiles["testprof"]
	if !ok {
		t.Fatal("testprof profile not found")
	}

	if p.Description != "test profile" {
		t.Errorf("Description = %q, want %q", p.Description, "test profile")
	}
	if p.Instances.PerCountry == nil || *p.Instances.PerCountry != 7 {
		t.Error("per_country should be 7")
	}
	if p.Instances.Countries == nil || *p.Instances.Countries != 3 {
		t.Error("countries should be 3")
	}
	if p.Relay.Enforce == nil || *p.Relay.Enforce != "exit" {
		t.Error("relay.enforce should be exit")
	}
	if p.Proxy.LoadBalanceAlgorithm == nil || *p.Proxy.LoadBalanceAlgorithm != "leastconn" {
		t.Error("proxy.load_balance_algorithm should be leastconn")
	}
	if p.Tor.MaxCircuitDirtiness == nil || *p.Tor.MaxCircuitDirtiness != 42 {
		t.Error("tor.max_circuit_dirtiness should be 42")
	}
	if p.Tor.ConnectionPadding == nil || *p.Tor.ConnectionPadding != 1 {
		t.Error("tor.connection_padding should be 1")
	}
	if p.Tor.UseEntryGuards == nil || *p.Tor.UseEntryGuards != 0 {
		t.Error("tor.use_entry_guards should be 0")
	}
	if p.Tor.StreamIsolation == nil || !*p.Tor.StreamIsolation {
		t.Error("tor.stream_isolation should be true")
	}
	if p.Tor.IPv6 == nil || !*p.Tor.IPv6 {
		t.Error("tor.ipv6 should be true")
	}
	if p.Tor.ConfluxEnabled == nil || !*p.Tor.ConfluxEnabled {
		t.Error("tor.conflux_enabled should be true")
	}
	if p.Tor.CongestionControlAuto == nil || !*p.Tor.CongestionControlAuto {
		t.Error("tor.congestion_control_auto should be true")
	}
	if p.Tor.Sandbox == nil || !*p.Tor.Sandbox {
		t.Error("tor.sandbox should be true")
	}
	if p.Tor.CircuitFingerprintingResistance == nil || !*p.Tor.CircuitFingerprintingResistance {
		t.Error("tor.circuit_fingerprinting_resistance should be true")
	}
	if p.Logging.Enabled == nil || !*p.Logging.Enabled {
		t.Error("logging.enabled should be true")
	}
	if p.Logging.Level == nil || *p.Logging.Level != "DEBUG" {
		t.Error("logging.level should be DEBUG")
	}
	if p.Country.RotationInterval == nil || *p.Country.RotationInterval != 90 {
		t.Error("country.rotation_interval should be 90")
	}
	if p.Country.TotalToChange == nil || *p.Country.TotalToChange != 3 {
		t.Error("country.total_to_change should be 3")
	}
	if p.HealthCheck.ExitReputation == nil || !*p.HealthCheck.ExitReputation {
		t.Error("health_check.exit_reputation should be true")
	}
}
