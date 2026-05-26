package cmd

import (
	"testing"

	"github.com/user/splitter/internal/config"
)

func TestDiffConfig_NoChanges(t *testing.T) {
	cfg := testReloadConfig(t)
	result := diffConfig(cfg, cfg)

	if result.CountryListChanged || result.RotationChanged || result.HAProxyChanged || result.TorChanged {
		t.Errorf("expected no changes, got %+v", result)
	}
	if result.Summary() != "no changes detected" {
		t.Errorf("Summary() = %q, want %q", result.Summary(), "no changes detected")
	}
}

func TestDiffConfig_CountryListAccepted(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Country.Accepted = []string{"{US}", "{DE}"}

	result := diffConfig(old, newCfg)
	if !result.CountryListChanged {
		t.Error("expected CountryListChanged")
	}
	if result.RotationChanged || result.HAProxyChanged || result.TorChanged {
		t.Errorf("unexpected changes: %+v", result)
	}
}

func TestDiffConfig_CountryListBlacklisted(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Country.Blacklisted = []string{"{RU}"}

	result := diffConfig(old, newCfg)
	if !result.CountryListChanged {
		t.Error("expected CountryListChanged")
	}
}

func TestDiffConfig_RotationInterval(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Country.Rotation.Interval = 300

	result := diffConfig(old, newCfg)
	if !result.RotationChanged {
		t.Error("expected RotationChanged")
	}
	if result.CountryListChanged || result.HAProxyChanged || result.TorChanged {
		t.Errorf("unexpected changes: %+v", result)
	}
}

func TestDiffConfig_RotationEnabled(t *testing.T) {
	old := testReloadConfig(t)
	old.Country.Rotation.Enabled = true
	newCfg := testReloadConfig(t)
	newCfg.Country.Rotation.Enabled = false

	result := diffConfig(old, newCfg)
	if !result.RotationChanged {
		t.Error("expected RotationChanged")
	}
}

func TestDiffConfig_RotationTotalToChange(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Country.Rotation.TotalToChange = 5

	result := diffConfig(old, newCfg)
	if !result.RotationChanged {
		t.Error("expected RotationChanged")
	}
}

func TestDiffConfig_HAProxyProxyChanged(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Proxy.LoadBalanceAlgorithm = "leastconn"

	result := diffConfig(old, newCfg)
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
	if result.CountryListChanged || result.RotationChanged || result.TorChanged {
		t.Errorf("unexpected changes: %+v", result)
	}
}

func TestDiffConfig_HAProxyHealthCheckChanged(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.HealthCheck.Interval = 30

	result := diffConfig(old, newCfg)
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
}

func TestDiffConfig_HAProxyRetriesChanged(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Instances.Retries = 500

	result := diffConfig(old, newCfg)
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
}

func TestDiffConfig_HAProxyProxyModeChanged(t *testing.T) {
	old := testReloadConfig(t)
	old.ProxyMode = "native"
	newCfg := testReloadConfig(t)
	newCfg.ProxyMode = "legacy"

	result := diffConfig(old, newCfg)
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
}

func TestDiffConfig_HAProxyPrivoxyChanged(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Privoxy.StartPort = 8000

	result := diffConfig(old, newCfg)
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
}

func TestDiffConfig_TorConfigChanged(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Tor.CircuitBuildTimeout = 120

	result := diffConfig(old, newCfg)
	if !result.TorChanged {
		t.Error("expected TorChanged")
	}
	if result.CountryListChanged || result.RotationChanged || result.HAProxyChanged {
		t.Errorf("unexpected changes: %+v", result)
	}
}

func TestDiffConfig_TorRelayEnforceChanged(t *testing.T) {
	old := testReloadConfig(t)
	old.Relay.Enforce = "entry"
	newCfg := testReloadConfig(t)
	newCfg.Relay.Enforce = "exit"

	result := diffConfig(old, newCfg)
	if !result.TorChanged {
		t.Error("expected TorChanged")
	}
}

func TestDiffConfig_MultipleChanges(t *testing.T) {
	old := testReloadConfig(t)
	newCfg := testReloadConfig(t)
	newCfg.Country.Accepted = []string{"{US}"}
	newCfg.Country.Rotation.Interval = 300
	newCfg.Proxy.LoadBalanceAlgorithm = "leastconn"
	newCfg.Tor.CircuitBuildTimeout = 120

	result := diffConfig(old, newCfg)
	if !result.CountryListChanged {
		t.Error("expected CountryListChanged")
	}
	if !result.RotationChanged {
		t.Error("expected RotationChanged")
	}
	if !result.HAProxyChanged {
		t.Error("expected HAProxyChanged")
	}
	if !result.TorChanged {
		t.Error("expected TorChanged")
	}
}

func TestConfigDiff_Summary_Multiple(t *testing.T) {
	d := configDiff{
		CountryListChanged: true,
		RotationChanged:    true,
	}
	summary := d.Summary()
	if summary != "country list, rotation interval" {
		t.Errorf("Summary() = %q, want %q", summary, "country list, rotation interval")
	}
}

func TestConfigDiff_Summary_All(t *testing.T) {
	d := configDiff{
		CountryListChanged: true,
		RotationChanged:    true,
		HAProxyChanged:     true,
		TorChanged:         true,
	}
	expected := "country list, rotation interval, haproxy config, tor config (restart needed)"
	if d.Summary() != expected {
		t.Errorf("Summary() = %q, want %q", d.Summary(), expected)
	}
}

func TestConfigDiff_Summary_None(t *testing.T) {
	d := configDiff{}
	if d.Summary() != "no changes detected" {
		t.Errorf("Summary() = %q, want %q", d.Summary(), "no changes detected")
	}
}

func testReloadConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.Defaults()
	return cfg
}
