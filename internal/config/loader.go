package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type FlagReader interface {
	Changed(name string) bool
	GetInt(name string) (int, error)
	GetString(name string) (string, error)
	GetBool(name string) (bool, error)
}

type LoadOptions struct {
	ConfigPath     string
	ProfilesPath   string
	UserAgentsPath string
	EnvPrefix      string
	Flags          FlagReader
}

func Load(opts LoadOptions) (*Config, error) {
	cfg := Defaults()

	if opts.ConfigPath == "" {
		opts.ConfigPath = "configs/default.yaml"
	}
	if opts.EnvPrefix == "" {
		opts.EnvPrefix = "SPLITTER_"
	}

	if err := loadYAML(cfg, opts.ConfigPath); err != nil {
		return nil, fmt.Errorf("Load: %w", err)
	}

	if err := loadUserAgents(cfg, opts.UserAgentsPath); err != nil {
		return nil, fmt.Errorf("Load: %w", err)
	}

	profileName := cfg.Profile
	if opts.Flags != nil && opts.Flags.Changed("profile") {
		if v, err := opts.Flags.GetString("profile"); err == nil {
			profileName = v
			cfg.Profile = v
		}
	}

	if err := applyProfile(cfg, profileName, opts.ProfilesPath); err != nil {
		return nil, fmt.Errorf("Load: %w", err)
	}

	if err := applyEnvOverrides(cfg, opts.EnvPrefix); err != nil {
		return nil, fmt.Errorf("Load: %w", err)
	}

	if opts.Flags != nil {
		applyFlags(cfg, opts.Flags)
	}

	if cfg.LogLevel != "" {
		cfg.Logging.Level = strings.ToUpper(cfg.LogLevel)
	}
	if cfg.Log {
		cfg.Logging.Enabled = true
	}

	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("Load: %w", err)
	}

	return cfg, nil
}

func loadYAML(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("loadYAML: reading %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("loadYAML: parsing %s: %w", path, err)
	}
	return nil
}

func applyFlags(cfg *Config, flags FlagReader) {
	if flags.Changed("instances") {
		if v, err := flags.GetInt("instances"); err == nil {
			cfg.Instances.PerCountry = v
		}
	}
	if flags.Changed("countries") {
		if v, err := flags.GetInt("countries"); err == nil {
			cfg.Instances.Countries = v
		}
	}
	if flags.Changed("relay-enforce") {
		if v, err := flags.GetString("relay-enforce"); err == nil {
			cfg.Relay.Enforce = v
		}
	}
	if flags.Changed("profile") {
		if v, err := flags.GetString("profile"); err == nil {
			cfg.Profile = v
		}
	}
	if flags.Changed("proxy-mode") {
		if v, err := flags.GetString("proxy-mode"); err == nil {
			cfg.ProxyMode = v
		}
	}
	if flags.Changed("bridge-type") {
		if v, err := flags.GetString("bridge-type"); err == nil {
			cfg.BridgeType = v
		}
	}
	if flags.Changed("verbose") {
		if v, err := flags.GetBool("verbose"); err == nil {
			cfg.Verbose = v
		}
	}
	if flags.Changed("log") {
		if v, err := flags.GetBool("log"); err == nil {
			cfg.Log = v
		}
	}
	if flags.Changed("log-level") {
		if v, err := flags.GetString("log-level"); err == nil {
			cfg.LogLevel = v
		}
	}
	if flags.Changed("auto-countries") {
		if v, err := flags.GetBool("auto-countries"); err == nil {
			cfg.Country.AutoCountries = v
		}
	}
	if flags.Changed("stream-isolation") {
		if v, err := flags.GetBool("stream-isolation"); err == nil {
			cfg.Tor.StreamIsolation = v
		}
	}
	if flags.Changed("ipv6") {
		if v, err := flags.GetBool("ipv6"); err == nil {
			cfg.Tor.IPv6 = v
		}
	}
	if flags.Changed("exit-reputation") {
		if v, err := flags.GetBool("exit-reputation"); err == nil {
			cfg.ExitReputation.Enabled = v
		}
	}
}

func loadUserAgents(cfg *Config, userAgentsPath string) error {
	if userAgentsPath == "" {
		userAgentsPath = "configs/useragents.yaml"
	}
	data, err := os.ReadFile(userAgentsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("loadUserAgents: %w", err)
	}

	var uaFile struct {
		UserAgents []string `yaml:"user_agents"`
		Default    string   `yaml:"default"`
	}
	if err := yaml.Unmarshal(data, &uaFile); err != nil {
		return fmt.Errorf("loadUserAgents: parsing: %w", err)
	}

	if len(uaFile.UserAgents) > 0 {
		cfg.UserAgent.UserAgents = uaFile.UserAgents
	}
	if uaFile.Default != "" {
		cfg.UserAgent.Default = uaFile.Default
		cfg.UserAgent.TorBrowser = uaFile.Default
	}
	return nil
}
