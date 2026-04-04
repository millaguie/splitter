package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type envSetter func(*Config, string)

var envMappings = map[string]envSetter{
	"INSTANCES":                         func(c *Config, v string) { c.Instances.PerCountry = atoi(v) },
	"INSTANCES_PER_COUNTRY":             func(c *Config, v string) { c.Instances.PerCountry = atoi(v) },
	"INSTANCES_COUNTRIES":               func(c *Config, v string) { c.Instances.Countries = atoi(v) },
	"INSTANCES_MAX_CONCURRENT_REQUESTS": func(c *Config, v string) { c.Instances.MaxConcurrentRequests = atoi(v) },
	"INSTANCES_RETRIES":                 func(c *Config, v string) { c.Instances.Retries = atoi(v) },
	"COUNTRIES":                         func(c *Config, v string) { c.Instances.Countries = atoi(v) },

	"RELAY_ENFORCE": func(c *Config, v string) { c.Relay.Enforce = v },

	"PROXY_MODE":                     func(c *Config, v string) { c.ProxyMode = v },
	"PROXY_LOAD_BALANCE_ALGORITHM":   func(c *Config, v string) { c.Proxy.LoadBalanceAlgorithm = v },
	"PROXY_MASTER_LISTEN":            func(c *Config, v string) { c.Proxy.Master.Listen = v },
	"PROXY_MASTER_PORT":              func(c *Config, v string) { c.Proxy.Master.Port = atoi(v) },
	"PROXY_MASTER_SOCKS_PORT":        func(c *Config, v string) { c.Proxy.Master.SocksPort = atoi(v) },
	"PROXY_MASTER_HTTP_PORT":         func(c *Config, v string) { c.Proxy.Master.HTTPPort = atoi(v) },
	"PROXY_MASTER_TRANSPARENT_PORT":  func(c *Config, v string) { c.Proxy.Master.TransparentPort = atoi(v) },
	"PROXY_MASTER_CLIENT_TIMEOUT":    func(c *Config, v string) { c.Proxy.Master.ClientTimeout = atoi(v) },
	"PROXY_MASTER_SERVER_TIMEOUT":    func(c *Config, v string) { c.Proxy.Master.ServerTimeout = atoi(v) },
	"PROXY_STATS_LISTEN":             func(c *Config, v string) { c.Proxy.Stats.Listen = v },
	"PROXY_STATS_PORT":               func(c *Config, v string) { c.Proxy.Stats.Port = atoi(v) },
	"PROXY_STATS_URI":                func(c *Config, v string) { c.Proxy.Stats.URI = v },
	"PROXY_HAPROXY_HTTP_REUSE":       func(c *Config, v string) { c.Proxy.HAProxyHTTPReuse = v },
	"PROXY_INCLUDE_SECURITY_HEADERS": func(c *Config, v string) { c.Proxy.IncludeSecurityHeaders = parseBool(v) },

	"TOR_BINARY_PATH":                       func(c *Config, v string) { c.Tor.BinaryPath = v },
	"TOR_LISTEN_ADDR":                       func(c *Config, v string) { c.Tor.ListenAddr = v },
	"TOR_START_SOCKS_PORT":                  func(c *Config, v string) { c.Tor.StartSocksPort = atoi(v) },
	"TOR_START_CONTROL_PORT":                func(c *Config, v string) { c.Tor.StartControlPort = atoi(v) },
	"TOR_START_HTTP_PORT":                   func(c *Config, v string) { c.Tor.StartHTTPPort = atoi(v) },
	"TOR_START_TRANSPORT_PORT":              func(c *Config, v string) { c.Tor.StartTransportPort = atoi(v) },
	"TOR_START_DNS_PORT":                    func(c *Config, v string) { c.Tor.StartDNSPort = atoi(v) },
	"TOR_CONTROL_AUTH":                      func(c *Config, v string) { c.Tor.ControlAuth = v },
	"TOR_MINIMUM_TIMEOUT":                   func(c *Config, v string) { c.Tor.MinimumTimeout = atoi(v) },
	"TOR_CIRCUIT_BUILD_TIMEOUT":             func(c *Config, v string) { c.Tor.CircuitBuildTimeout = atoi(v) },
	"TOR_LEARN_CIRCUIT_BUILD_TIMEOUT":       func(c *Config, v string) { c.Tor.LearnCircuitBuildTimeout = atoi(v) },
	"TOR_CIRCUITS_AVAILABLE_TIMEOUT":        func(c *Config, v string) { c.Tor.CircuitsAvailableTimeout = atoi(v) },
	"TOR_CIRCUIT_STREAM_TIMEOUT":            func(c *Config, v string) { c.Tor.CircuitStreamTimeout = atoi(v) },
	"TOR_CLIENT_ONLY":                       func(c *Config, v string) { c.Tor.ClientOnly = atoi(v) },
	"TOR_CONNECTION_PADDING":                func(c *Config, v string) { c.Tor.ConnectionPadding = atoi(v) },
	"TOR_REDUCED_CONNECTION_PADDING":        func(c *Config, v string) { c.Tor.ReducedConnectionPadding = atoi(v) },
	"TOR_GEOIP_EXCLUDE_UNKNOWN":             func(c *Config, v string) { c.Tor.GeoIPExcludeUnknown = atoi(v) },
	"TOR_STRICT_NODES":                      func(c *Config, v string) { c.Tor.StrictNodes = atoi(v) },
	"TOR_FASCIST_FIREWALL":                  func(c *Config, v string) { c.Tor.FascistFirewall = atoi(v) },
	"TOR_NEW_CIRCUIT_PERIOD":                func(c *Config, v string) { c.Tor.NewCircuitPeriod = atoi(v) },
	"TOR_MAX_CIRCUIT_DIRTINESS":             func(c *Config, v string) { c.Tor.MaxCircuitDirtiness = atoi(v) },
	"TOR_MAX_CLIENT_CIRCUITS_PENDING":       func(c *Config, v string) { c.Tor.MaxClientCircuitsPending = atoi(v) },
	"TOR_SOCKS_TIMEOUT":                     func(c *Config, v string) { c.Tor.SocksTimeout = atoi(v) },
	"TOR_TRACK_HOST_EXITS_EXPIRE":           func(c *Config, v string) { c.Tor.TrackHostExitsExpire = atoi(v) },
	"TOR_USE_ENTRY_GUARDS":                  func(c *Config, v string) { c.Tor.UseEntryGuards = atoi(v) },
	"TOR_NUM_ENTRY_GUARDS":                  func(c *Config, v string) { c.Tor.NumEntryGuards = atoi(v) },
	"TOR_SAFE_SOCKS":                        func(c *Config, v string) { c.Tor.SafeSocks = atoi(v) },
	"TOR_TEST_SOCKS":                        func(c *Config, v string) { c.Tor.TestSocks = atoi(v) },
	"TOR_OPTIMISTIC_DATA":                   func(c *Config, v string) { c.Tor.OptimisticData = v },
	"TOR_AUTOMAP_HOSTS_SUFFIXES":            func(c *Config, v string) { c.Tor.AutomapHostsSuffixes = v },
	"TOR_WARN_PLAINTEXT_PORTS":              func(c *Config, v string) { c.Tor.WarnPlaintextPorts = v },
	"TOR_REJECT_PLAINTEXT_PORTS":            func(c *Config, v string) { c.Tor.RejectPlaintextPorts = v },
	"TOR_STREAM_ISOLATION":                  func(c *Config, v string) { c.Tor.StreamIsolation = parseBool(v) },
	"TOR_IPV6":                              func(c *Config, v string) { c.Tor.IPv6 = parseBool(v) },
	"TOR_CONFLUX_ENABLED":                   func(c *Config, v string) { c.Tor.ConfluxEnabled = parseBool(v) },
	"TOR_CONGESTION_CONTROL_AUTO":           func(c *Config, v string) { c.Tor.CongestionControlAuto = parseBool(v) },
	"TOR_CIRCUIT_FINGERPRINTING_RESISTANCE": func(c *Config, v string) { c.Tor.CircuitFingerprintingResistance = parseBool(v) },
	"TOR_SANDBOX":                           func(c *Config, v string) { c.Tor.Sandbox = parseBool(v) },

	"PRIVOXY_BINARY_PATH":        func(c *Config, v string) { c.Privoxy.BinaryPath = v },
	"PRIVOXY_LISTEN":             func(c *Config, v string) { c.Privoxy.Listen = v },
	"PRIVOXY_START_PORT":         func(c *Config, v string) { c.Privoxy.StartPort = atoi(v) },
	"PRIVOXY_TIMEOUT":            func(c *Config, v string) { c.Privoxy.Timeout = atoi(v) },
	"PRIVOXY_CONFIG_FILE_PREFIX": func(c *Config, v string) { c.Privoxy.ConfigFilePrefix = v },

	"HAPROXY_BINARY_PATH": func(c *Config, v string) { c.HAProxy.BinaryPath = v },
	"HAPROXY_CONFIG_FILE": func(c *Config, v string) { c.HAProxy.ConfigFile = v },

	"COUNTRY_SELECTED":                 func(c *Config, v string) { c.Country.Selected = v },
	"COUNTRY_ROTATION_ENABLED":         func(c *Config, v string) { c.Country.Rotation.Enabled = parseBool(v) },
	"COUNTRY_ROTATION_INTERVAL":        func(c *Config, v string) { c.Country.Rotation.Interval = atoi(v) },
	"COUNTRY_ROTATION_TOTAL_TO_CHANGE": func(c *Config, v string) { c.Country.Rotation.TotalToChange = atoi(v) },
	"COUNTRY_AUTO_COUNTRIES":           func(c *Config, v string) { c.Country.AutoCountries = parseBool(v) },

	"HEALTH_CHECK_URL":             func(c *Config, v string) { c.HealthCheck.URL = v },
	"HEALTH_CHECK_INTERVAL":        func(c *Config, v string) { c.HealthCheck.Interval = atoi(v) },
	"HEALTH_CHECK_MAX_FAIL":        func(c *Config, v string) { c.HealthCheck.MaxFail = atoi(v) },
	"HEALTH_CHECK_MINIMUM_SUCCESS": func(c *Config, v string) { c.HealthCheck.MinimumSuccess = atoi(v) },

	"USER_AGENT_TOR_BROWSER": func(c *Config, v string) { c.UserAgent.TorBrowser = v },
	"USER_AGENT_DEFAULT":     func(c *Config, v string) { c.UserAgent.Default = v },

	"LOG":             func(c *Config, v string) { c.Logging.Enabled = parseBool(v); c.Log = parseBool(v) },
	"LOG_LEVEL":       func(c *Config, v string) { c.Logging.Level = v; c.LogLevel = v },
	"LOGGING_ENABLED": func(c *Config, v string) { c.Logging.Enabled = parseBool(v) },
	"LOGGING_DIR":     func(c *Config, v string) { c.Logging.Dir = v },
	"LOGGING_LEVEL":   func(c *Config, v string) { c.Logging.Level = v },
	"LOGGING_FORMAT":  func(c *Config, v string) { c.Logging.Format = v },

	"PATHS_TEMP_FILES":       func(c *Config, v string) { c.Paths.TempFiles = v },
	"PATHS_PROXYCHAINS_FILE": func(c *Config, v string) { c.Paths.ProxychainsFile = v },

	"DNS_DIST_LISTEN": func(c *Config, v string) { c.DNS.DistListen = v },
	"DNS_DIST_PORT":   func(c *Config, v string) { c.DNS.DistPort = atoi(v) },
	"DNS_TOR_LISTEN":  func(c *Config, v string) { c.DNS.TorListen = v },

	"EXIT_REPUTATION": func(c *Config, v string) { c.ExitReputation.Enabled = parseBool(v) },

	"PROFILE":     func(c *Config, v string) { c.Profile = v },
	"BRIDGE_TYPE": func(c *Config, v string) { c.BridgeType = v },
	"VERBOSE":     func(c *Config, v string) { c.Verbose = parseBool(v) },
}

func applyEnvOverrides(cfg *Config, prefix string) error {
	envVars := os.Environ()
	upperPrefix := strings.ToUpper(prefix)
	if !strings.HasSuffix(upperPrefix, "_") {
		upperPrefix += "_"
	}

	var errs []string
	for _, envVar := range envVars {
		parts := strings.SplitN(envVar, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		if !strings.HasPrefix(key, upperPrefix) {
			continue
		}

		suffix := strings.TrimPrefix(key, upperPrefix)
		setter, ok := envMappings[suffix]
		if !ok {
			continue
		}
		setter(cfg, value)
	}

	if len(errs) > 0 {
		return fmt.Errorf("applyEnvOverrides: %s", strings.Join(errs, "; "))
	}
	return nil
}

func atoi(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}

func parseBool(s string) bool {
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "1" || s == "true" || s == "yes" || s == "on"
}
