package config

import (
	"testing"
)

func TestValidate_EntryMode(t *testing.T) {
	cfg := Defaults()
	cfg.Relay.Enforce = "entry"
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with entry mode error = %v", err)
	}
}

func TestValidate_ExitMode(t *testing.T) {
	cfg := Defaults()
	cfg.Relay.Enforce = "exit"
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with exit mode error = %v", err)
	}
}

func TestValidate_SpeedMode(t *testing.T) {
	cfg := Defaults()
	cfg.Relay.Enforce = "speed"
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with speed mode error = %v", err)
	}
}

func TestValidate_PortZero(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 0
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with port 0 should be valid, got error = %v", err)
	}
}

func TestValidate_PortOne(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 1
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with port 1 should be valid, got error = %v", err)
	}
}

func TestValidate_Port65534(t *testing.T) {
	// Port 65534 with 1 instance: 65534 + 1 = 65535, which is <= 65535, valid.
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 65534
	cfg.Instances.PerCountry = 1
	cfg.Instances.Countries = 1
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with port 65534+1 should be valid, got error = %v", err)
	}
}

func TestValidate_Port65535Overflow(t *testing.T) {
	// Port 65535 with 1 instance: 65535 + 1 = 65536 > 65535, should fail.
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 65535
	cfg.Instances.PerCountry = 1
	cfg.Instances.Countries = 1
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with port 65535 + 1 instance should fail (65536 > 65535)")
	}
}

func TestValidate_PortAbove65535(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartSocksPort = 65536
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with port 65536 expected error, got nil")
	}
}

func TestValidate_PortNegative(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartSocksPort = -1
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with negative port expected error, got nil")
	}
}

func TestValidate_AllPortsAtZero(t *testing.T) {
	cfg := Defaults()
	cfg.Proxy.Master.Port = 0
	cfg.Proxy.Master.SocksPort = 0
	cfg.Proxy.Master.HTTPPort = 0
	cfg.Proxy.Master.TransparentPort = 0
	cfg.Proxy.Stats.Port = 0
	cfg.Tor.StartSocksPort = 0
	cfg.Tor.StartControlPort = 0
	cfg.Tor.StartHTTPPort = 0
	cfg.Tor.StartTransportPort = 0
	cfg.Tor.StartDNSPort = 0
	cfg.Tor.HiddenService.StartPort = 0
	cfg.Privoxy.StartPort = 0
	cfg.DNS.DistPort = 0
	if err := Validate(cfg); err != nil {
		t.Errorf("Validate() with all ports at 0 should be valid, got error = %v", err)
	}
}

func TestValidate_InvalidLogLevel(t *testing.T) {
	cfg := Defaults()
	cfg.LogLevel = "trace"
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with log level trace expected error")
	}
}

func TestValidate_ValidLogLevels(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error"} {
		t.Run(level, func(t *testing.T) {
			cfg := Defaults()
			cfg.LogLevel = level
			if err := Validate(cfg); err != nil {
				t.Errorf("Validate() with log level %q error = %v", level, err)
			}
		})
	}
}

func TestValidate_BridgeTypes(t *testing.T) {
	tests := []struct {
		bridge string
		valid  bool
	}{
		{"snowflake", true},
		{"webtunnel", true},
		{"obfs4", true},
		{"none", true},
		{"meek", false},
		{"custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.bridge, func(t *testing.T) {
			cfg := Defaults()
			cfg.BridgeType = tt.bridge
			err := Validate(cfg)
			if (err == nil) != tt.valid {
				t.Errorf("Validate() with bridge=%q valid=%v, got err=%v", tt.bridge, tt.valid, err)
			}
		})
	}
}

func TestValidate_ProxyModes(t *testing.T) {
	tests := []struct {
		mode  string
		valid bool
	}{
		{"native", true},
		{"legacy", true},
		{"socks5", false},
		{"http", false},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := Defaults()
			cfg.ProxyMode = tt.mode
			err := Validate(cfg)
			if (err == nil) != tt.valid {
				t.Errorf("Validate() with proxy_mode=%q valid=%v, got err=%v", tt.mode, tt.valid, err)
			}
		})
	}
}

func TestValidate_Profiles(t *testing.T) {
	tests := []struct {
		profile string
		valid   bool
	}{
		{"stealth", true},
		{"balanced", true},
		{"streaming", true},
		{"pentest", true},
		{"", true},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.profile, func(t *testing.T) {
			cfg := Defaults()
			cfg.Profile = tt.profile
			err := Validate(cfg)
			if (err == nil) != tt.valid {
				t.Errorf("Validate() with profile=%q valid=%v, got err=%v", tt.profile, tt.valid, err)
			}
		})
	}
}

func TestInSet(t *testing.T) {
	if !inSet("entry", "entry", "exit", "speed") {
		t.Error("inSet(entry) should be true")
	}
	if inSet("invalid", "entry", "exit", "speed") {
		t.Error("inSet(invalid) should be false")
	}
	if !inSet("speed", "entry", "exit", "speed") {
		t.Error("inSet(speed) should be true")
	}
}

func TestValidate_StatsPortAbove65535(t *testing.T) {
	cfg := Defaults()
	cfg.Proxy.Stats.Port = 70000
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with stats port 70000 expected error")
	}
}

func TestValidate_ControlPortNegative(t *testing.T) {
	cfg := Defaults()
	cfg.Tor.StartControlPort = -5
	if err := Validate(cfg); err == nil {
		t.Error("Validate() with negative control port expected error")
	}
}
