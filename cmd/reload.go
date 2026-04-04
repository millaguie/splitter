package cmd

import (
	"reflect"
	"strings"

	"github.com/user/splitter/internal/config"
)

type configDiff struct {
	CountryListChanged bool
	RotationChanged    bool
	HAProxyChanged     bool
	TorChanged         bool
}

func diffConfig(old, newCfg *config.Config) configDiff {
	var d configDiff

	if !reflect.DeepEqual(old.Country.Accepted, newCfg.Country.Accepted) ||
		!reflect.DeepEqual(old.Country.Blacklisted, newCfg.Country.Blacklisted) {
		d.CountryListChanged = true
	}

	if old.Country.Rotation.Interval != newCfg.Country.Rotation.Interval ||
		old.Country.Rotation.Enabled != newCfg.Country.Rotation.Enabled ||
		old.Country.Rotation.TotalToChange != newCfg.Country.Rotation.TotalToChange {
		d.RotationChanged = true
	}

	if !reflect.DeepEqual(old.Proxy, newCfg.Proxy) ||
		!reflect.DeepEqual(old.HealthCheck, newCfg.HealthCheck) ||
		old.Instances.Retries != newCfg.Instances.Retries ||
		old.ProxyMode != newCfg.ProxyMode ||
		!reflect.DeepEqual(old.Privoxy, newCfg.Privoxy) {
		d.HAProxyChanged = true
	}

	if !reflect.DeepEqual(old.Tor, newCfg.Tor) ||
		old.Relay.Enforce != newCfg.Relay.Enforce {
		d.TorChanged = true
	}

	return d
}

func (d configDiff) Summary() string {
	var parts []string
	if d.CountryListChanged {
		parts = append(parts, "country list")
	}
	if d.RotationChanged {
		parts = append(parts, "rotation interval")
	}
	if d.HAProxyChanged {
		parts = append(parts, "haproxy config")
	}
	if d.TorChanged {
		parts = append(parts, "tor config (restart needed)")
	}
	if len(parts) == 0 {
		return "no changes detected"
	}
	return strings.Join(parts, ", ")
}
