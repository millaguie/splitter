package config

import (
	"fmt"
	"strings"
)

func Validate(cfg *Config) error {
	if cfg.Instances.PerCountry <= 0 {
		return fmt.Errorf("validate: instances.per_country must be > 0, got %d", cfg.Instances.PerCountry)
	}
	if cfg.Instances.Countries <= 0 {
		return fmt.Errorf("validate: instances.countries must be > 0, got %d", cfg.Instances.Countries)
	}

	if !inSet(cfg.Relay.Enforce, "entry", "exit", "speed") {
		return fmt.Errorf("validate: relay.enforce must be entry|exit|speed, got %q", cfg.Relay.Enforce)
	}
	if !inSet(cfg.ProxyMode, "native", "legacy") {
		return fmt.Errorf("validate: proxy_mode must be native|legacy, got %q", cfg.ProxyMode)
	}
	if !inSet(cfg.BridgeType, "snowflake", "webtunnel", "obfs4", "none") {
		return fmt.Errorf("validate: bridge_type must be snowflake|webtunnel|obfs4|none, got %q", cfg.BridgeType)
	}
	if cfg.Profile != "" && !inSet(cfg.Profile, "stealth", "balanced", "streaming", "pentest") {
		return fmt.Errorf("validate: profile must be stealth|balanced|streaming|pentest, got %q", cfg.Profile)
	}
	if cfg.LogLevel != "" && !inSet(strings.ToLower(cfg.LogLevel), "debug", "info", "warn", "error") {
		return fmt.Errorf("validate: log_level must be debug|info|warn|error, got %q", cfg.LogLevel)
	}
	if !inSet(strings.ToLower(cfg.Logging.Level), "debug", "info", "warn", "error") {
		return fmt.Errorf("validate: logging.level must be debug|info|warn|error, got %q", cfg.Logging.Level)
	}

	if err := validatePorts(cfg); err != nil {
		return err
	}

	totalPorts := cfg.Instances.PerCountry * cfg.Instances.Countries
	maxPort := cfg.Tor.StartSocksPort + totalPorts
	if maxPort > 65535 {
		return fmt.Errorf("validate: SOCKS port range exceeds 65535 (start=%d + %d instances = %d)",
			cfg.Tor.StartSocksPort, totalPorts, maxPort)
	}

	return nil
}

func validatePorts(cfg *Config) error {
	ports := []struct {
		name string
		val  int
	}{
		{"proxy.master.port", cfg.Proxy.Master.Port},
		{"proxy.master.socks_port", cfg.Proxy.Master.SocksPort},
		{"proxy.master.http_port", cfg.Proxy.Master.HTTPPort},
		{"proxy.master.transparent_port", cfg.Proxy.Master.TransparentPort},
		{"proxy.stats.port", cfg.Proxy.Stats.Port},
		{"tor.start_socks_port", cfg.Tor.StartSocksPort},
		{"tor.start_control_port", cfg.Tor.StartControlPort},
		{"tor.start_http_port", cfg.Tor.StartHTTPPort},
		{"tor.start_transport_port", cfg.Tor.StartTransportPort},
		{"tor.start_dns_port", cfg.Tor.StartDNSPort},
		{"tor.hidden_service.start_port", cfg.Tor.HiddenService.StartPort},
		{"privoxy.start_port", cfg.Privoxy.StartPort},
		{"dns.dist_port", cfg.DNS.DistPort},
	}

	for _, p := range ports {
		if p.val < 0 || p.val > 65535 {
			return fmt.Errorf("validate: %s must be 0-65535, got %d", p.name, p.val)
		}
	}
	return nil
}

func inSet(val string, valid ...string) bool {
	for _, v := range valid {
		if val == v {
			return true
		}
	}
	return false
}
