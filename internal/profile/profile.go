package profile

const (
	Stealth   = "stealth"
	Balanced  = "balanced"
	Streaming = "streaming"
	Pentest   = "pentest"
)

var ValidProfiles = []string{Stealth, Balanced, Streaming, Pentest}

type Profile struct {
	Description string             `yaml:"description"`
	Instances   ProfileInstances   `yaml:"instances"`
	Relay       ProfileRelay       `yaml:"relay"`
	Proxy       ProfileProxy       `yaml:"proxy"`
	Tor         ProfileTor         `yaml:"tor"`
	Logging     ProfileLogging     `yaml:"logging"`
	Country     ProfileCountry     `yaml:"country"`
	HealthCheck ProfileHealthCheck `yaml:"health_check"`
}

type ProfileInstances struct {
	PerCountry            *int `yaml:"per_country"`
	Countries             *int `yaml:"countries"`
	MaxConcurrentRequests *int `yaml:"max_concurrent_requests"`
	Retries               *int `yaml:"retries"`
}

type ProfileRelay struct {
	Enforce *string `yaml:"enforce"`
}

type ProfileProxy struct {
	LoadBalanceAlgorithm *string `yaml:"load_balance_algorithm"`
	HAProxyHTTPReuse     *string `yaml:"haproxy_http_reuse"`
}

type ProfileTor struct {
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

type ProfileLogging struct {
	Enabled *bool   `yaml:"enabled"`
	Level   *string `yaml:"level"`
}

type ProfileCountry struct {
	RotationInterval *int `yaml:"rotation_interval"`
	TotalToChange    *int `yaml:"total_to_change"`
}

type ProfileHealthCheck struct {
	ExitReputation *bool `yaml:"exit_reputation"`
}

func IsValid(name string) bool {
	for _, p := range ValidProfiles {
		if name == p {
			return true
		}
	}
	return false
}

func Names() []string {
	return append([]string{}, ValidProfiles...)
}
