package config

import (
	"testing"
)

// TestConfigDefaults_PrivacySettings verifies that the compiled-in default
// configuration has privacy-safe values. If any of these defaults are changed
// to insecure values, these tests will catch it.
func TestConfigDefaults_PrivacySettings(t *testing.T) {
	cfg := Defaults()

	tests := []struct {
		name     string
		got      int
		want     int
		insecure string // description of what insecure means
	}{
		{
			name:     "safe_socks must be 1 (prevents DNS leaks via SOCKS)",
			got:      cfg.Tor.SafeSocks,
			want:     1,
			insecure: "DNS can leak via SOCKS hostname resolution",
		},
		{
			name:     "strict_nodes must be 1 (never fall back to unlisted nodes)",
			got:      cfg.Tor.StrictNodes,
			want:     1,
			insecure: "Traffic may use nodes outside selected countries",
		},
		{
			name:     "client_reject_internal_addresses must be 1",
			got:      cfg.Tor.ClientRejectInternalAddresses,
			want:     1,
			insecure: "Connections to internal/private IPs would be allowed",
		},
		{
			name:     "geoip_exclude_unknown must be 1",
			got:      cfg.Tor.GeoIPExcludeUnknown,
			want:     1,
			insecure: "Nodes from unknown jurisdictions can be selected",
		},
		{
			name:     "test_socks must be 1 (log SOCKS safety rejections)",
			got:      cfg.Tor.TestSocks,
			want:     1,
			insecure: "SOCKS protocol violations go unlogged",
		},
		{
			name:     "client_only must be 0 (no relay, client-only)",
			got:      cfg.Tor.ClientOnly,
			want:     0,
			insecure: "Running as relay exposes your IP as a Tor node",
		},
		{
			name:     "allow_non_rfc953_hostnames must be 0",
			got:      cfg.Tor.AllowNonRFC953Hostnames,
			want:     0,
			insecure: "Malformed hostnames could cause DNS leaks",
		},
		{
			name:     "download_extra_info must be 0 (reduce bandwidth fingerprint)",
			got:      cfg.Tor.DownloadExtraInfo,
			want:     0,
			insecure: "Extra downloads increase bandwidth fingerprint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("default = %d, want %d (SECURITY: %s)", tt.got, tt.want, tt.insecure)
			}
		})
	}
}

// TestConfigDefaults_PrivacyStrings verifies string-valued privacy defaults.
func TestConfigDefaults_PrivacyStrings(t *testing.T) {
	cfg := Defaults()

	if cfg.Tor.AutomapHostsSuffixes == "" {
		t.Error("automap_hosts_suffixes is empty — .onion resolution will not work")
	}
	if cfg.Tor.WarnPlaintextPorts == "" {
		t.Error("warn_plaintext_ports is empty — users won't be warned about plaintext traffic")
	}
}

// TestConfigDefaults_ControlAuth verifies control port auth defaults.
func TestConfigDefaults_ControlAuth(t *testing.T) {
	cfg := Defaults()

	// Cookie auth should be a supported value
	validAuth := map[string]bool{
		"cookie":   true,
		"password": true,
	}
	if !validAuth[cfg.Tor.ControlAuth] {
		t.Errorf("control_auth = %q, want 'cookie' or 'password'", cfg.Tor.ControlAuth)
	}
}

// TestConfigDefaults_ReducedConnectionPadding verifies padding defaults
// for traffic analysis resistance.
func TestConfigDefaults_ReducedConnectionPadding(t *testing.T) {
	cfg := Defaults()

	// Reduced padding should be on by default (saves bandwidth while
	// maintaining some padding protection)
	if cfg.Tor.ReducedConnectionPadding != 1 {
		t.Errorf("reduced_connection_padding = %d, want 1", cfg.Tor.ReducedConnectionPadding)
	}
}

// TestConfigDefaults_EntryGuards verifies entry guard defaults for
// long-term entry node protection.
func TestConfigDefaults_EntryGuards(t *testing.T) {
	cfg := Defaults()

	if cfg.Tor.UseEntryGuards != 1 {
		t.Errorf("use_entry_guards = %d, want 1 (SECURITY: entry nodes change frequently without guards)", cfg.Tor.UseEntryGuards)
	}
	if cfg.Tor.NumEntryGuards < 1 {
		t.Errorf("num_entry_guards = %d, want >= 1", cfg.Tor.NumEntryGuards)
	}
}

// TestConfigDefaults_CircuitTimeouts verifies circuit timeout defaults
// are reasonable for privacy (not too short, not too long).
func TestConfigDefaults_CircuitTimeouts(t *testing.T) {
	cfg := Defaults()

	if cfg.Tor.CircuitBuildTimeout < 30 {
		t.Errorf("circuit_build_timeout = %d, want >= 30 (too short causes circuit failures)", cfg.Tor.CircuitBuildTimeout)
	}
	if cfg.Tor.NewCircuitPeriod < 10 {
		t.Errorf("new_circuit_period = %d, want >= 10 (too short causes excessive circuit building)", cfg.Tor.NewCircuitPeriod)
	}
	if cfg.Tor.MaxCircuitDirtiness < 10 {
		t.Errorf("max_circuit_dirtiness = %d, want >= 10 (too short reduces circuit reuse)", cfg.Tor.MaxCircuitDirtiness)
	}
	if cfg.Tor.MaxClientCircuitsPending < 32 {
		t.Errorf("max_client_circuits_pending = %d, want >= 32 (too low limits parallel connections)", cfg.Tor.MaxClientCircuitsPending)
	}
}

// TestConfigDefaults_StreamIsolationDefault verifies stream isolation
// is off by default (it's a tradeoff: more isolation vs more circuits).
func TestConfigDefaults_StreamIsolationDefault(t *testing.T) {
	cfg := Defaults()

	if cfg.Tor.StreamIsolation {
		t.Error("stream_isolation should be false by default (enable explicitly or via pentest profile)")
	}
}
