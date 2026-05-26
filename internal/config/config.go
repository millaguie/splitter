package config

import "math/rand"

type Config struct {
	Instances      InstancesConfig      `yaml:"instances"`
	Proxy          ProxyConfig          `yaml:"proxy"`
	Relay          RelayConfig          `yaml:"relay"`
	Tor            TorConfig            `yaml:"tor"`
	Privoxy        PrivoxyConfig        `yaml:"privoxy"`
	HAProxy        HAProxyConfig        `yaml:"haproxy"`
	Country        CountryConfig        `yaml:"country"`
	HealthCheck    HealthCheckConfig    `yaml:"health_check"`
	UserAgent      UserAgentConfig      `yaml:"user_agent"`
	Logging        LoggingConfig        `yaml:"logging"`
	Paths          PathsConfig          `yaml:"paths"`
	DNS            DNSConfig            `yaml:"dns"`
	ExitReputation ExitReputationConfig `yaml:"exit_reputation"`

	Profile    string
	ProxyMode  string
	BridgeType string
	Verbose    bool
	Log        bool
	LogLevel   string
}

type InstancesConfig struct {
	PerCountry            int `yaml:"per_country"`
	Countries             int `yaml:"countries"`
	MaxConcurrentRequests int `yaml:"max_concurrent_requests"`
	Retries               int `yaml:"retries"`
}

type ProxyConfig struct {
	Master                 ProxyMasterConfig `yaml:"master"`
	Stats                  ProxyStatsConfig  `yaml:"stats"`
	LoadBalanceAlgorithm   string            `yaml:"load_balance_algorithm"`
	HAProxyHTTPReuse       string            `yaml:"haproxy_http_reuse"`
	IncludeSecurityHeaders bool              `yaml:"include_security_headers"`
	DoNotProxy             []string          `yaml:"do_not_proxy"`
}

type ProxyMasterConfig struct {
	Listen          string `yaml:"listen"`
	Port            int    `yaml:"port"`
	SocksPort       int    `yaml:"socks_port"`
	HTTPPort        int    `yaml:"http_port"`
	TransparentPort int    `yaml:"transparent_port"`
	ClientTimeout   int    `yaml:"client_timeout"`
	ServerTimeout   int    `yaml:"server_timeout"`
}

type ProxyStatsConfig struct {
	Listen string `yaml:"listen"`
	Port   int    `yaml:"port"`
	URI    string `yaml:"uri"`
}

type RelayConfig struct {
	Enforce string `yaml:"enforce"`
}

type TorConfig struct {
	BinaryPath                      string                 `yaml:"binary_path"`
	ListenAddr                      string                 `yaml:"listen_addr"`
	StartSocksPort                  int                    `yaml:"start_socks_port"`
	StartControlPort                int                    `yaml:"start_control_port"`
	StartHTTPPort                   int                    `yaml:"start_http_port"`
	StartTransportPort              int                    `yaml:"start_transport_port"`
	StartDNSPort                    int                    `yaml:"start_dns_port"`
	ControlAuth                     string                 `yaml:"control_auth"`
	HiddenService                   TorHiddenServiceConfig `yaml:"hidden_service"`
	MinimumTimeout                  int                    `yaml:"minimum_timeout"`
	CircuitBuildTimeout             int                    `yaml:"circuit_build_timeout"`
	LearnCircuitBuildTimeout        int                    `yaml:"learn_circuit_build_timeout"`
	CircuitsAvailableTimeout        int                    `yaml:"circuits_available_timeout"`
	CircuitStreamTimeout            int                    `yaml:"circuit_stream_timeout"`
	ClientOnly                      int                    `yaml:"client_only"`
	ConnectionPadding               int                    `yaml:"connection_padding"`
	ReducedConnectionPadding        int                    `yaml:"reduced_connection_padding"`
	GeoIPExcludeUnknown             int                    `yaml:"geoip_exclude_unknown"`
	StrictNodes                     int                    `yaml:"strict_nodes"`
	FascistFirewall                 int                    `yaml:"fascist_firewall"`
	FirewallPorts                   []int                  `yaml:"firewall_ports"`
	LongLivedPorts                  []int                  `yaml:"long_lived_ports"`
	NewCircuitPeriod                int                    `yaml:"new_circuit_period"`
	MaxCircuitDirtiness             int                    `yaml:"max_circuit_dirtiness"`
	MaxClientCircuitsPending        int                    `yaml:"max_client_circuits_pending"`
	SocksTimeout                    int                    `yaml:"socks_timeout"`
	TrackHostExitsExpire            int                    `yaml:"track_host_exits_expire"`
	UseEntryGuards                  int                    `yaml:"use_entry_guards"`
	NumEntryGuards                  int                    `yaml:"num_entry_guards"`
	SafeSocks                       int                    `yaml:"safe_socks"`
	TestSocks                       int                    `yaml:"test_socks"`
	AllowNonRFC953Hostnames         int                    `yaml:"allow_non_rfc953_hostnames"`
	ClientRejectInternalAddresses   int                    `yaml:"client_reject_internal_addresses"`
	DownloadExtraInfo               int                    `yaml:"download_extra_info"`
	OptimisticData                  string                 `yaml:"optimistic_data"`
	AutomapHostsSuffixes            string                 `yaml:"automap_hosts_suffixes"`
	WarnPlaintextPorts              string                 `yaml:"warn_plaintext_ports"`
	RejectPlaintextPorts            string                 `yaml:"reject_plaintext_ports"`
	Sandbox                         bool                   `yaml:"sandbox"`
	StreamIsolation                 bool                   `yaml:"stream_isolation"`
	IPv6                            bool                   `yaml:"ipv6"`
	ConfluxEnabled                  bool                   `yaml:"conflux_enabled"`
	CongestionControlAuto           bool                   `yaml:"congestion_control_auto"`
	CircuitFingerprintingResistance bool                   `yaml:"circuit_fingerprinting_resistance"`
}

type TorHiddenServiceConfig struct {
	Enabled                bool   `yaml:"enabled"`
	BasePath               string `yaml:"base_path"`
	StartPort              int    `yaml:"start_port"`
	MaxStreams             int    `yaml:"max_streams"`
	MaxStreamsCloseCircuit bool   `yaml:"max_streams_close_circuit"`
	DirGroupReadable       bool   `yaml:"dir_group_readable"`
	NumIntroductionPoints  int    `yaml:"num_introduction_points"`
}

type PrivoxyConfig struct {
	BinaryPath       string `yaml:"binary_path"`
	Listen           string `yaml:"listen"`
	StartPort        int    `yaml:"start_port"`
	Timeout          int    `yaml:"timeout"`
	ConfigFilePrefix string `yaml:"config_file_prefix"`
}

type HAProxyConfig struct {
	BinaryPath string `yaml:"binary_path"`
	ConfigFile string `yaml:"config_file"`
}

type CountryConfig struct {
	Selected      string                `yaml:"selected"`
	Accepted      []string              `yaml:"accepted"`
	Blacklisted   []string              `yaml:"blacklisted"`
	Rotation      CountryRotationConfig `yaml:"rotation"`
	AutoCountries bool                  `yaml:"auto_countries"`
}

type CountryRotationConfig struct {
	Enabled       bool `yaml:"enabled"`
	Interval      int  `yaml:"interval"`
	TotalToChange int  `yaml:"total_to_change"`
}

type HealthCheckConfig struct {
	URL            string `yaml:"url"`
	Interval       int    `yaml:"interval"`
	MaxFail        int    `yaml:"max_fail"`
	MinimumSuccess int    `yaml:"minimum_success"`
}

type UserAgentConfig struct {
	TorBrowser string   `yaml:"tor_browser"`
	UserAgents []string `yaml:"user_agents"`
	Default    string   `yaml:"default"`
}

type LoggingConfig struct {
	Enabled    bool   `yaml:"enabled"`
	Dir        string `yaml:"dir"`
	NamePrefix string `yaml:"name_prefix"`
	Level      string `yaml:"level"`
	Format     string `yaml:"format"`
}

type PathsConfig struct {
	TempFiles       string `yaml:"temp_files"`
	ProxychainsFile string `yaml:"proxychains_file"`
}

type DNSConfig struct {
	DistListen string `yaml:"dist_listen"`
	DistPort   int    `yaml:"dist_port"`
	TorListen  string `yaml:"tor_listen"`
}

type ExitReputationConfig struct {
	Enabled bool `yaml:"enabled"`
}

func Defaults() *Config {
	return &Config{
		Instances: InstancesConfig{
			PerCountry:            2,
			Countries:             6,
			MaxConcurrentRequests: 20,
			Retries:               1000,
		},
		Proxy: ProxyConfig{
			Master: ProxyMasterConfig{
				Listen:          "0.0.0.0",
				Port:            63536,
				SocksPort:       63536,
				HTTPPort:        63537,
				TransparentPort: 63538,
				ClientTimeout:   35,
				ServerTimeout:   35,
			},
			Stats: ProxyStatsConfig{
				Listen: "0.0.0.0",
				Port:   63539,
				URI:    "/splitter_status",
			},
			LoadBalanceAlgorithm:   "roundrobin",
			HAProxyHTTPReuse:       "never",
			IncludeSecurityHeaders: true,
			DoNotProxy: []string{
				"0.0.0.0", "192.168.1.1", "192.168.2.1",
				"192.168.3.1", "192.168.0.1", "172.17.0.1",
			},
		},
		Relay: RelayConfig{
			Enforce: "entry",
		},
		Tor: TorConfig{
			BinaryPath:         "/usr/bin/tor",
			ListenAddr:         "0.0.0.0",
			StartSocksPort:     4999,
			StartControlPort:   5999,
			StartHTTPPort:      5199,
			StartTransportPort: 5099,
			StartDNSPort:       5299,
			ControlAuth:        "password",
			HiddenService: TorHiddenServiceConfig{
				Enabled:                true,
				BasePath:               "/tmp/splitter/hidden_service_",
				StartPort:              3999,
				MaxStreams:             0,
				MaxStreamsCloseCircuit: false,
				DirGroupReadable:       false,
				NumIntroductionPoints:  3,
			},
			MinimumTimeout:                15,
			CircuitBuildTimeout:           60,
			LearnCircuitBuildTimeout:      1,
			CircuitsAvailableTimeout:      5,
			CircuitStreamTimeout:          20,
			ClientOnly:                    0,
			ConnectionPadding:             0,
			ReducedConnectionPadding:      1,
			GeoIPExcludeUnknown:           1,
			StrictNodes:                   1,
			FascistFirewall:               0,
			FirewallPorts:                 []int{80, 443},
			LongLivedPorts:                []int{1, 2},
			NewCircuitPeriod:              30,
			MaxCircuitDirtiness:           15,
			MaxClientCircuitsPending:      1024,
			SocksTimeout:                  35,
			TrackHostExitsExpire:          10,
			UseEntryGuards:                1,
			NumEntryGuards:                1,
			SafeSocks:                     1,
			TestSocks:                     1,
			AllowNonRFC953Hostnames:       0,
			ClientRejectInternalAddresses: 1,
			DownloadExtraInfo:             0,
			OptimisticData:                "auto",
			AutomapHostsSuffixes:          ".exit,.onion",
			WarnPlaintextPorts:            "21,23,25,80,109,110,143",
			RejectPlaintextPorts:          "",
			IPv6:                          false,
		},
		Privoxy: PrivoxyConfig{
			BinaryPath:       "/usr/sbin/privoxy",
			Listen:           "0.0.0.0",
			StartPort:        6999,
			Timeout:          35,
			ConfigFilePrefix: "/tmp/splitter/privoxy_splitter_config_",
		},
		HAProxy: HAProxyConfig{
			BinaryPath: "/usr/sbin/haproxy",
			ConfigFile: "/tmp/splitter/splitter_master_proxy.cfg",
		},
		Country: CountryConfig{
			Selected: "RANDOM",
			Accepted: []string{
				"{AU}", "{AT}", "{BE}", "{BG}", "{CA}", "{CZ}", "{DK}",
				"{FI}", "{FR}", "{DE}", "{HU}", "{IS}", "{LV}", "{LT}",
				"{LU}", "{MD}", "{NL}", "{NO}", "{PA}", "{PL}", "{RO}",
				"{RU}", "{SC}", "{SG}", "{SK}", "{ES}", "{SE}", "{CH}",
				"{TR}", "{UA}", "{GB}", "{US}",
			},
			Rotation: CountryRotationConfig{
				Enabled:       true,
				Interval:      120,
				TotalToChange: 10,
			},
		},
		HealthCheck: HealthCheckConfig{
			URL:            "https://www.google.com/",
			Interval:       12,
			MaxFail:        1,
			MinimumSuccess: 1,
		},
		UserAgent: UserAgentConfig{
			TorBrowser: "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0",
			UserAgents: nil,
			Default:    "Mozilla/5.0 (Windows NT 10.0; rv:128.0) Gecko/20100101 Firefox/128.0",
		},
		Logging: LoggingConfig{
			Enabled:    false,
			Dir:        "/tmp/splitter",
			NamePrefix: "tor_log_",
			Level:      "INFO",
			Format:     "text",
		},
		Paths: PathsConfig{
			TempFiles:       "/tmp/splitter",
			ProxychainsFile: "",
		},
		DNS: DNSConfig{
			DistListen: "0.0.0.0",
			DistPort:   5353,
			TorListen:  "0.0.0.0",
		},
		ExitReputation: ExitReputationConfig{
			Enabled: false,
		},
		Profile:    "",
		ProxyMode:  "native",
		BridgeType: "none",
		Verbose:    false,
		Log:        false,
		LogLevel:   "",
	}
}

func (u *UserAgentConfig) PickUserAgent() string {
	if len(u.UserAgents) > 0 {
		return u.UserAgents[rand.Intn(len(u.UserAgents))]
	}
	if u.Default != "" {
		return u.Default
	}
	return u.TorBrowser
}
