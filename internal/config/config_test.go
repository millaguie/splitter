package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_DefaultsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "missing.yaml")

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_X_",
	})
	if err != nil {
		t.Fatalf("Load with missing file: %v", err)
	}

	if cfg.Instances.PerCountry != 2 {
		t.Errorf("PerCountry = %d, want 2", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries != 6 {
		t.Errorf("Countries = %d, want 6", cfg.Instances.Countries)
	}
	if cfg.Relay.Enforce != "entry" {
		t.Errorf("Enforce = %q, want %q", cfg.Relay.Enforce, "entry")
	}
	if cfg.ProxyMode != "native" {
		t.Errorf("ProxyMode = %q, want %q", cfg.ProxyMode, "native")
	}
	if cfg.BridgeType != "none" {
		t.Errorf("BridgeType = %q, want %q", cfg.BridgeType, "none")
	}
}

func TestLoad_YAMLFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 5
  countries: 3
relay:
  enforce: "speed"
tor:
  max_circuit_dirtiness: 60
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_YAML_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Instances.PerCountry != 5 {
		t.Errorf("PerCountry = %d, want 5", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries != 3 {
		t.Errorf("Countries = %d, want 3", cfg.Instances.Countries)
	}
	if cfg.Relay.Enforce != "speed" {
		t.Errorf("Enforce = %q, want %q", cfg.Relay.Enforce, "speed")
	}
	if cfg.Tor.MaxCircuitDirtiness != 60 {
		t.Errorf("MaxCircuitDirtiness = %d, want 60", cfg.Tor.MaxCircuitDirtiness)
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPLITTER_TEST_INSTANCES", "10")
	t.Setenv("SPLITTER_TEST_COUNTRIES", "12")
	t.Setenv("SPLITTER_TEST_RELAY_ENFORCE", "exit")
	t.Setenv("SPLITTER_TEST_PROXY_MODE", "legacy")

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Instances.PerCountry != 10 {
		t.Errorf("PerCountry = %d, want 10 (env override)", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries != 12 {
		t.Errorf("Countries = %d, want 12 (env override)", cfg.Instances.Countries)
	}
	if cfg.Relay.Enforce != "exit" {
		t.Errorf("Enforce = %q, want %q (env override)", cfg.Relay.Enforce, "exit")
	}
	if cfg.ProxyMode != "legacy" {
		t.Errorf("ProxyMode = %q, want %q (env override)", cfg.ProxyMode, "legacy")
	}
}

func TestLoad_FlagOverrides(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPLITTER_TEST_FLAG_INSTANCES", "10")

	flags := &stubFlagReader{
		changed: map[string]bool{
			"instances":     true,
			"countries":     true,
			"relay-enforce": true,
		},
		ints: map[string]int{
			"instances": 7,
			"countries": 4,
		},
		strings: map[string]string{
			"relay-enforce": "speed",
		},
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_FLAG_",
		Flags:      flags,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Instances.PerCountry != 7 {
		t.Errorf("PerCountry = %d, want 7 (flag override)", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries != 4 {
		t.Errorf("Countries = %d, want 4 (flag override)", cfg.Instances.Countries)
	}
	if cfg.Relay.Enforce != "speed" {
		t.Errorf("Enforce = %q, want %q (flag override)", cfg.Relay.Enforce, "speed")
	}
}

func TestLoad_PriorityOrder(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPLITTER_TEST_PR_INSTANCES", "10")

	flags := &stubFlagReader{
		changed: map[string]bool{"instances": true},
		ints:    map[string]int{"instances": 99},
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_PR_",
		Flags:      flags,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Instances.PerCountry != 99 {
		t.Errorf("PerCountry = %d, want 99 (flag > env > yaml)", cfg.Instances.PerCountry)
	}
}

func TestLoad_Profile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")
	profilesPath := filepath.Join(tmpDir, "profiles.yaml")

	yamlContent := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
`
	if err := os.WriteFile(cfgPath, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	profileContent := `
stealth:
  description: "test stealth"
  instances:
    per_country: 3
    countries: 8
  relay:
    enforce: "entry"
  tor:
    max_circuit_dirtiness: 10
    connection_padding: 1
`
	if err := os.WriteFile(profilesPath, []byte(profileContent), 0644); err != nil {
		t.Fatal(err)
	}

	flags := &stubFlagReader{
		changed: map[string]bool{"profile": true},
		strings: map[string]string{"profile": "stealth"},
	}

	cfg, err := Load(LoadOptions{
		ConfigPath:   cfgPath,
		ProfilesPath: profilesPath,
		EnvPrefix:    "SPLITTER_TEST_PROF_",
		Flags:        flags,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Instances.PerCountry != 3 {
		t.Errorf("PerCountry = %d, want 3 (stealth profile)", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries != 8 {
		t.Errorf("Countries = %d, want 8 (stealth profile)", cfg.Instances.Countries)
	}
	if cfg.Tor.MaxCircuitDirtiness != 10 {
		t.Errorf("MaxCircuitDirtiness = %d, want 10 (stealth profile)", cfg.Tor.MaxCircuitDirtiness)
	}
	if cfg.Tor.ConnectionPadding != 1 {
		t.Errorf("ConnectionPadding = %d, want 1 (stealth profile)", cfg.Tor.ConnectionPadding)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "default config is valid",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name:    "zero per_country",
			modify:  func(c *Config) { c.Instances.PerCountry = 0 },
			wantErr: true,
		},
		{
			name:    "negative per_country",
			modify:  func(c *Config) { c.Instances.PerCountry = -1 },
			wantErr: true,
		},
		{
			name:    "zero countries",
			modify:  func(c *Config) { c.Instances.Countries = 0 },
			wantErr: true,
		},
		{
			name:    "invalid relay enforce",
			modify:  func(c *Config) { c.Relay.Enforce = "invalid" },
			wantErr: true,
		},
		{
			name:    "invalid proxy mode",
			modify:  func(c *Config) { c.ProxyMode = "socks" },
			wantErr: true,
		},
		{
			name:    "invalid bridge type",
			modify:  func(c *Config) { c.BridgeType = "meek" },
			wantErr: true,
		},
		{
			name:    "invalid profile",
			modify:  func(c *Config) { c.Profile = "unknown" },
			wantErr: true,
		},
		{
			name:    "invalid log level",
			modify:  func(c *Config) { c.Logging.Level = "trace" },
			wantErr: true,
		},
		{
			name:    "port out of range",
			modify:  func(c *Config) { c.Proxy.Master.Port = 70000 },
			wantErr: true,
		},
		{
			name:    "port negative",
			modify:  func(c *Config) { c.Tor.StartSocksPort = -1 },
			wantErr: true,
		},
		{
			name:    "valid exit mode",
			modify:  func(c *Config) { c.Relay.Enforce = "exit" },
			wantErr: false,
		},
		{
			name:    "valid speed mode",
			modify:  func(c *Config) { c.Relay.Enforce = "speed" },
			wantErr: false,
		},
		{
			name:    "empty profile is valid",
			modify:  func(c *Config) { c.Profile = "" },
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.modify(cfg)
			err := Validate(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidate_SOCKSPortOverflow(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 65000
	cfg.Instances.PerCountry = 100
	cfg.Instances.Countries = 10

	err := Validate(cfg)
	if err == nil {
		t.Error("Validate() expected error for port overflow, got nil")
	}
}

func TestApplyEnvOverrides_BoolParsing(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"1", true},
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"yes", true},
		{"on", true},
		{"0", false},
		{"false", false},
		{"no", false},
	}

	for _, tt := range tests {
		t.Run(tt.val, func(t *testing.T) {
			cfg := Defaults()
			cfg.Logging.Enabled = false
			t.Setenv("SPLITTER_TEST_BOOL_LOGGING_ENABLED", tt.val)
			_ = applyEnvOverrides(cfg, "SPLITTER_TEST_BOOL_")
			if cfg.Logging.Enabled != tt.want {
				t.Errorf("parseBool(%q) = %v, want %v", tt.val, cfg.Logging.Enabled, tt.want)
			}
		})
	}
}

func TestLoad_YAMLMalformed(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "bad.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances: [broken yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_MALFORMED_",
	})
	if err == nil {
		t.Error("Load() expected error for malformed YAML, got nil")
	}
}

func TestLoad_UnknownProfile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")
	profilesPath := filepath.Join(tmpDir, "profiles.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilesPath, []byte("stealth:\n  description: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	flags := &stubFlagReader{
		changed: map[string]bool{"profile": true},
		strings: map[string]string{"profile": "nonexistent"},
	}

	_, err := Load(LoadOptions{
		ConfigPath:   cfgPath,
		ProfilesPath: profilesPath,
		EnvPrefix:    "SPLITTER_TEST_UNKPROF_",
		Flags:        flags,
	})
	if err == nil {
		t.Error("Load() expected error for unknown profile, got nil")
	}
}

func TestLoad_LogFlagEnablesLogging(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	flags := &stubFlagReader{
		changed: map[string]bool{"log": true},
		bools:   map[string]bool{"log": true},
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_LOGFLAG_",
		Flags:      flags,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Logging.Enabled {
		t.Error("Logging.Enabled should be true when log flag is set")
	}
}

func TestLoad_StreamIsolation_Default(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_SI_DEFAULT_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Tor.StreamIsolation {
		t.Error("StreamIsolation should be false by default")
	}
}

func TestLoad_StreamIsolation_YAML(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
tor:
  stream_isolation: true
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_SI_YAML_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.Tor.StreamIsolation {
		t.Error("StreamIsolation should be true from YAML")
	}
}

func TestLoad_StreamIsolation_EnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
tor:
  stream_isolation: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPLITTER_TEST_SI_ENV_TOR_STREAM_ISOLATION", "true")

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_SI_ENV_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.Tor.StreamIsolation {
		t.Error("StreamIsolation should be true from env override")
	}
}

func TestLoad_StreamIsolation_FlagOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	content := `
instances:
  per_country: 2
  countries: 6
relay:
  enforce: "entry"
tor:
  stream_isolation: false
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	flags := &stubFlagReader{
		changed: map[string]bool{"stream-isolation": true},
		bools:   map[string]bool{"stream-isolation": true},
	}

	cfg, err := Load(LoadOptions{
		ConfigPath: cfgPath,
		EnvPrefix:  "SPLITTER_TEST_SI_FLAG_",
		Flags:      flags,
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if !cfg.Tor.StreamIsolation {
		t.Error("StreamIsolation should be true from flag override")
	}
}

type stubFlagReader struct {
	changed map[string]bool
	ints    map[string]int
	strings map[string]string
	bools   map[string]bool
}

func (s *stubFlagReader) Changed(name string) bool {
	return s.changed[name]
}

func (s *stubFlagReader) GetInt(name string) (int, error) {
	if v, ok := s.ints[name]; ok {
		return v, nil
	}
	return 0, nil
}

func (s *stubFlagReader) GetString(name string) (string, error) {
	if v, ok := s.strings[name]; ok {
		return v, nil
	}
	return "", nil
}

func (s *stubFlagReader) GetBool(name string) (bool, error) {
	if v, ok := s.bools[name]; ok {
		return v, nil
	}
	return false, nil
}

func TestLoadUserAgents_FromFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")
	uaPath := filepath.Join(tmpDir, "useragents.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	uaContent := `
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0"
  - "Mozilla/5.0 (X11; Linux x86_64; rv:115.0) Gecko/20100101 Firefox/115.0"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:102.0) Gecko/20100101 Firefox/102.0"
default: "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0"
`
	if err := os.WriteFile(uaPath, []byte(uaContent), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigPath:     cfgPath,
		UserAgentsPath: uaPath,
		EnvPrefix:      "SPLITTER_TEST_UA_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if len(cfg.UserAgent.UserAgents) != 3 {
		t.Errorf("UserAgents length = %d, want 3", len(cfg.UserAgent.UserAgents))
	}
	if cfg.UserAgent.Default != "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0" {
		t.Errorf("Default = %q, want Tor Browser 128 UA", cfg.UserAgent.Default)
	}
	if cfg.UserAgent.TorBrowser != "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0" {
		t.Errorf("TorBrowser = %q, want updated to default UA", cfg.UserAgent.TorBrowser)
	}
}

func TestLoadUserAgents_MissingFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(LoadOptions{
		ConfigPath:     cfgPath,
		UserAgentsPath: filepath.Join(tmpDir, "nonexistent.yaml"),
		EnvPrefix:      "SPLITTER_TEST_UA_MISS_",
	})
	if err != nil {
		t.Fatalf("Load with missing useragents file: %v", err)
	}

	if len(cfg.UserAgent.UserAgents) != 0 {
		t.Errorf("UserAgents length = %d, want 0 (file missing)", len(cfg.UserAgent.UserAgents))
	}
	if cfg.UserAgent.Default != "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0" {
		t.Errorf("Default = %q, want built-in default", cfg.UserAgent.Default)
	}
}

func TestPickUserAgent_FromList(t *testing.T) {
	uas := []string{
		"Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0",
		"Mozilla/5.0 (X11; Linux x86_64; rv:115.0) Gecko/20100101 Firefox/115.0",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:102.0) Gecko/20100101 Firefox/102.0",
	}
	ua := &UserAgentConfig{
		TorBrowser: "fallback",
		UserAgents: uas,
		Default:    "default-ua",
	}

	for i := 0; i < 50; i++ {
		picked := ua.PickUserAgent()
		found := false
		for _, u := range uas {
			if picked == u {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PickUserAgent() = %q, not in user_agents list", picked)
		}
	}
}

func TestPickUserAgent_EmptyList(t *testing.T) {
	ua := &UserAgentConfig{
		TorBrowser: "fallback-ua",
		UserAgents: nil,
		Default:    "default-ua",
	}

	picked := ua.PickUserAgent()
	if picked != "default-ua" {
		t.Errorf("PickUserAgent() = %q, want %q (Default fallback)", picked, "default-ua")
	}
}

func TestPickUserAgent_EmptyListNoDefault(t *testing.T) {
	ua := &UserAgentConfig{
		TorBrowser: "legacy-ua",
		UserAgents: nil,
		Default:    "",
	}

	picked := ua.PickUserAgent()
	if picked != "legacy-ua" {
		t.Errorf("PickUserAgent() = %q, want %q (TorBrowser fallback)", picked, "legacy-ua")
	}
}

func TestPickUserAgent_EmptySliceUsesDefault(t *testing.T) {
	ua := &UserAgentConfig{
		TorBrowser: "legacy-ua",
		UserAgents: []string{},
		Default:    "default-ua",
	}

	picked := ua.PickUserAgent()
	if picked != "default-ua" {
		t.Errorf("PickUserAgent() = %q, want %q (empty slice uses Default)", picked, "default-ua")
	}
}

func TestLoadUserAgents_EnvOverride(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "test.yaml")
	uaPath := filepath.Join(tmpDir, "useragents.yaml")

	if err := os.WriteFile(cfgPath, []byte("instances:\n  per_country: 2\n  countries: 6\nrelay:\n  enforce: entry\n"), 0644); err != nil {
		t.Fatal(err)
	}

	uaContent := `
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0"
default: "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0"
`
	if err := os.WriteFile(uaPath, []byte(uaContent), 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("SPLITTER_TEST_UAENV_USER_AGENT_DEFAULT", "custom-ua-from-env")

	cfg, err := Load(LoadOptions{
		ConfigPath:     cfgPath,
		UserAgentsPath: uaPath,
		EnvPrefix:      "SPLITTER_TEST_UAENV_",
	})
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.UserAgent.Default != "custom-ua-from-env" {
		t.Errorf("Default = %q, want %q (env override)", cfg.UserAgent.Default, "custom-ua-from-env")
	}
}
