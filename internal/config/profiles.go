package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type profileEntry struct {
	Description string              `yaml:"description"`
	Instances   *profileInstances   `yaml:"instances"`
	Relay       *profileRelay       `yaml:"relay"`
	Proxy       *profileProxy       `yaml:"proxy"`
	Tor         *profileTor         `yaml:"tor"`
	Logging     *profileLogging     `yaml:"logging"`
	Country     *profileCountry     `yaml:"country"`
	HealthCheck *profileHealthCheck `yaml:"health_check"`
}

type profileInstances struct {
	PerCountry            *int `yaml:"per_country"`
	Countries             *int `yaml:"countries"`
	MaxConcurrentRequests *int `yaml:"max_concurrent_requests"`
	Retries               *int `yaml:"retries"`
}

type profileRelay struct {
	Enforce *string `yaml:"enforce"`
}

type profileProxy struct {
	LoadBalanceAlgorithm *string `yaml:"load_balance_algorithm"`
	HAProxyHTTPReuse     *string `yaml:"haproxy_http_reuse"`
}

type profileTor struct {
	MaxCircuitDirtiness             *int  `yaml:"max_circuit_dirtiness"`
	ConnectionPadding               *int  `yaml:"connection_padding"`
	UseEntryGuards                  *int  `yaml:"use_entry_guards"`
	ReducedConnectionPadding        *int  `yaml:"reduced_connection_padding"`
	StreamIsolation                 *bool `yaml:"stream_isolation"`
	IPv6                            *bool `yaml:"ipv6"`
	ConfluxEnabled                  *bool `yaml:"conflux_enabled"`
	CongestionControlAuto           *bool `yaml:"congestion_control_auto"`
	Sandbox                         *bool `yaml:"sandbox"`
	CircuitFingerprintingResistance *bool `yaml:"circuit_fingerprinting_resistance"`
}

type profileLogging struct {
	Enabled *bool   `yaml:"enabled"`
	Level   *string `yaml:"level"`
}

type profileCountry struct {
	RotationInterval *int `yaml:"rotation_interval"`
	TotalToChange    *int `yaml:"total_to_change"`
}

type profileHealthCheck struct {
	ExitReputation *bool `yaml:"exit_reputation"`
}

func applyProfile(cfg *Config, profileName string, profilesPath string) error {
	if profileName == "" {
		return nil
	}

	if profilesPath == "" {
		profilesPath = "configs/profiles.yaml"
	}

	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return fmt.Errorf("applyProfile: reading %s: %w", profilesPath, err)
	}

	var profiles map[string]profileEntry
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return fmt.Errorf("applyProfile: parsing %s: %w", profilesPath, err)
	}

	entry, ok := profiles[profileName]
	if !ok {
		return fmt.Errorf("applyProfile: unknown profile %q", profileName)
	}

	if entry.Instances != nil {
		if entry.Instances.PerCountry != nil {
			cfg.Instances.PerCountry = *entry.Instances.PerCountry
		}
		if entry.Instances.Countries != nil {
			cfg.Instances.Countries = *entry.Instances.Countries
		}
		if entry.Instances.MaxConcurrentRequests != nil {
			cfg.Instances.MaxConcurrentRequests = *entry.Instances.MaxConcurrentRequests
		}
		if entry.Instances.Retries != nil {
			cfg.Instances.Retries = *entry.Instances.Retries
		}
	}

	if entry.Relay != nil && entry.Relay.Enforce != nil {
		cfg.Relay.Enforce = *entry.Relay.Enforce
	}

	if entry.Proxy != nil {
		if entry.Proxy.LoadBalanceAlgorithm != nil {
			cfg.Proxy.LoadBalanceAlgorithm = *entry.Proxy.LoadBalanceAlgorithm
		}
		if entry.Proxy.HAProxyHTTPReuse != nil {
			cfg.Proxy.HAProxyHTTPReuse = *entry.Proxy.HAProxyHTTPReuse
		}
	}

	if entry.Tor != nil {
		if entry.Tor.MaxCircuitDirtiness != nil {
			cfg.Tor.MaxCircuitDirtiness = *entry.Tor.MaxCircuitDirtiness
		}
		if entry.Tor.ConnectionPadding != nil {
			cfg.Tor.ConnectionPadding = *entry.Tor.ConnectionPadding
		}
		if entry.Tor.UseEntryGuards != nil {
			cfg.Tor.UseEntryGuards = *entry.Tor.UseEntryGuards
		}
		if entry.Tor.ReducedConnectionPadding != nil {
			cfg.Tor.ReducedConnectionPadding = *entry.Tor.ReducedConnectionPadding
		}
		if entry.Tor.StreamIsolation != nil {
			cfg.Tor.StreamIsolation = *entry.Tor.StreamIsolation
		}
		if entry.Tor.IPv6 != nil {
			cfg.Tor.IPv6 = *entry.Tor.IPv6
		}
		if entry.Tor.ConfluxEnabled != nil {
			cfg.Tor.ConfluxEnabled = *entry.Tor.ConfluxEnabled
		}
		if entry.Tor.CongestionControlAuto != nil {
			cfg.Tor.CongestionControlAuto = *entry.Tor.CongestionControlAuto
		}
		if entry.Tor.Sandbox != nil {
			cfg.Tor.Sandbox = *entry.Tor.Sandbox
		}
		if entry.Tor.CircuitFingerprintingResistance != nil {
			cfg.Tor.CircuitFingerprintingResistance = *entry.Tor.CircuitFingerprintingResistance
		}
	}

	if entry.Logging != nil {
		if entry.Logging.Enabled != nil {
			cfg.Logging.Enabled = *entry.Logging.Enabled
		}
		if entry.Logging.Level != nil {
			cfg.Logging.Level = *entry.Logging.Level
		}
	}

	if entry.Country != nil {
		if entry.Country.RotationInterval != nil {
			cfg.Country.Rotation.Interval = *entry.Country.RotationInterval
		}
		if entry.Country.TotalToChange != nil {
			cfg.Country.Rotation.TotalToChange = *entry.Country.TotalToChange
		}
	}

	if entry.HealthCheck != nil {
		if entry.HealthCheck.ExitReputation != nil {
			cfg.ExitReputation.Enabled = *entry.HealthCheck.ExitReputation
		}
	}

	return nil
}
