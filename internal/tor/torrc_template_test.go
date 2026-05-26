package tor

import (
	"os"
	"strings"
	"testing"
)

func baseInstanceConfig() InstanceConfig {
	return InstanceConfig{
		InstanceID:                    0,
		Country:                       "{US}",
		SocksPort:                     9050,
		ControlPort:                   9051,
		HTTPTunnelPort:                0,
		DataDir:                       "/tmp/test",
		CircuitBuildTimeout:           60,
		CircuitStreamTimeout:          20,
		MaxCircuitDirtiness:           30,
		NewCircuitPeriod:              30,
		LearnCircuitBuildTimeout:      1,
		CongestionControlAuto:         false,
		ConfluxEnabled:                false,
		RelayEnforce:                  "entry",
		HiddenServiceEnabled:          true,
		HiddenServiceDir:              "/tmp/hs",
		HiddenServicePort:             8080,
		ConnectionPadding:             0,
		ReducedConnectionPadding:      1,
		SafeSocks:                     1,
		TestSocks:                     1,
		ClientRejectInternalAddresses: 1,
		StrictNodes:                   1,
		ClientOnly:                    0,
		GeoIPExcludeUnknown:           1,
		FascistFirewall:               0,
		FirewallPorts:                 []int{80, 443},
		LongLivedPorts:                []int{1, 2},
		MaxClientCircuitsPending:      1024,
		SocksTimeout:                  35,
		TrackHostExitsExpire:          10,
		UseEntryGuards:                1,
		NumEntryGuards:                1,
		AutomapHostsSuffixes:          ".exit,.onion",
		WarnPlaintextPorts:            "21,23,25",
		RejectPlaintextPorts:          "",
		KeepalivePeriod:               15,
		ControlAuth:                   "cookie",
		StreamIsolation:               false,
		ClientUseIPv6:                 false,
	}
}

func readTorrcTemplate(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile("../../templates/torrc.gotmpl")
	if err != nil {
		t.Fatalf("read torrc template: %v", err)
	}
	return string(data)
}

func torrcContains(t *testing.T, result, substr string) {
	t.Helper()
	if !strings.Contains(result, substr) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", substr, result)
	}
}

func torrcNotContains(t *testing.T, result, substr string) {
	t.Helper()
	if strings.Contains(result, substr) {
		t.Errorf("expected output NOT to contain %q\nfull output:\n%s", substr, result)
	}
}

func TestTorrcTemplate_EntryMode(t *testing.T) {
	ic := baseInstanceConfig()
	ic.RelayEnforce = "entry"
	ic.Country = "{US}"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "EntryNodes {US}")
	torrcNotContains(t, result, "ExitNodes")
}

func TestTorrcTemplate_ExitMode(t *testing.T) {
	ic := baseInstanceConfig()
	ic.RelayEnforce = "exit"
	ic.Country = "{DE}"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "ExitNodes {DE}")
	torrcNotContains(t, result, "EntryNodes")
}

func TestTorrcTemplate_SpeedMode(t *testing.T) {
	ic := baseInstanceConfig()
	ic.RelayEnforce = "speed"
	ic.Country = "{FR}"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "EntryNodes")
	torrcNotContains(t, result, "ExitNodes")
	torrcContains(t, result, "speed mode")
}

func TestTorrcTemplate_HTTP_TUNNEL_Port(t *testing.T) {
	ic := baseInstanceConfig()
	ic.HTTPTunnelPort = 5199

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "HTTPTunnelPort 5199")
}

func TestTorrcTemplate_NoHTTP_TUNNEL_Port(t *testing.T) {
	ic := baseInstanceConfig()
	ic.HTTPTunnelPort = 0

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "HTTPTunnelPort")
}

func TestTorrcTemplate_CongestionControl(t *testing.T) {
	ic := baseInstanceConfig()
	ic.CongestionControlAuto = true

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "CongestionControlAuto 1")
}

func TestTorrcTemplate_NoCongestionControl(t *testing.T) {
	ic := baseInstanceConfig()
	ic.CongestionControlAuto = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "CongestionControlAuto")
}

func TestTorrcTemplate_ConfluxEnabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.ConfluxEnabled = true

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "ConfluxEnabled 1")
}

func TestTorrcTemplate_NoConflux(t *testing.T) {
	ic := baseInstanceConfig()
	ic.ConfluxEnabled = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "ConfluxEnabled")
}

func TestTorrcTemplate_HiddenService(t *testing.T) {
	ic := baseInstanceConfig()
	ic.HiddenServiceEnabled = true
	ic.HiddenServiceDir = "/tmp/hs"
	ic.HiddenServicePort = 8080

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "HiddenServiceDir /tmp/hs")
	torrcContains(t, result, "HiddenServicePort 8080")
}

func TestTorrcTemplate_NoHiddenService(t *testing.T) {
	ic := baseInstanceConfig()
	ic.HiddenServiceEnabled = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "HiddenServiceDir")
	torrcNotContains(t, result, "HiddenServicePort")
}

func TestTorrcTemplate_FascistFirewall(t *testing.T) {
	ic := baseInstanceConfig()
	ic.FascistFirewall = 1
	ic.FirewallPorts = []int{80, 443}

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "FascistFirewall 1")
	torrcContains(t, result, "FirewallPorts 80,443")
}

func TestTorrcTemplate_RejectPlaintextPorts(t *testing.T) {
	ic := baseInstanceConfig()
	ic.RejectPlaintextPorts = "25,119"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "RejectPlaintextPorts 25,119")
}

func TestTorrcTemplate_CookieAuthentication(t *testing.T) {
	ic := baseInstanceConfig()

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "CookieAuthentication 1")
}

func TestTorrcTemplate_SandboxEnabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.SandboxEnabled = true

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "Sandbox 1")
	torrcNotContains(t, result, "Sandbox: disabled")
}

func TestTorrcTemplate_SandboxDisabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.SandboxEnabled = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcNotContains(t, result, "Sandbox 1")
	torrcContains(t, result, "Sandbox: disabled")
}

func TestTorrcTemplate_StreamIsolationEnabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.StreamIsolation = true

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "SocksPort 9050 IsolateSOCKSAuth")
}

func TestTorrcTemplate_StreamIsolationDisabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.StreamIsolation = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "SocksPort 9050")
	torrcNotContains(t, result, "SocksPort 9050 IsolateSOCKSAuth")
}

func TestTorrcTemplate_IPv6Enabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.ClientUseIPv6 = true

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "ClientUseIPv6 1")
	torrcNotContains(t, result, "ClientUseIPv6 0")
}

func TestTorrcTemplate_IPv6Disabled(t *testing.T) {
	ic := baseInstanceConfig()
	ic.ClientUseIPv6 = false

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "ClientUseIPv6 0")
	torrcNotContains(t, result, "ClientUseIPv6 1")
}
