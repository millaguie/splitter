package tor

import (
	"strings"
	"testing"
)

// TestTorrcTemplate_HardcodedSecurityOptions verifies that security-critical
// torrc directives that are hardcoded (not configurable) are always present
// in the rendered output. These prevent template regressions from silently
// disabling security features.
func TestTorrcTemplate_HardcodedSecurityOptions(t *testing.T) {
	ic := baseInstanceConfig()
	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	// SafeLogging: prevents Tor from logging potentially sensitive data
	torrcContains(t, result, "SafeLogging 1")

	// NoExec: prevents Tor from executing external programs (attack surface)
	torrcContains(t, result, "NoExec 1")

	// DisableDebuggerAttachment: prevents debugger attachment to Tor process
	torrcContains(t, result, "DisableDebuggerAttachment 1")

	// EnforceDistinctSubnets: prevents multiple relays in the same /16 subnet
	torrcContains(t, result, "EnforceDistinctSubnets 1")

	// ClientUseIPv4: ensures IPv4 is always enabled
	torrcContains(t, result, "ClientUseIPv4 1")

	// RunAsDaemon 0: Tor runs in foreground (managed by SPLITTER)
	torrcContains(t, result, "RunAsDaemon 0")
}

// TestTorrcTemplate_ConfigurableSecurityOptions verifies that configurable
// security options from InstanceConfig appear correctly in rendered torrc.
// These are set in config but could be overridden to insecure values.
func TestTorrcTemplate_ConfigurableSecurityOptions(t *testing.T) {
	ic := baseInstanceConfig()
	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	// DNS leak prevention: reject SOCKS requests with hostnames that resolve
	// to private/internal addresses
	torrcContains(t, result, "SafeSocks 1")

	// Log when SOCKS safety checks reject a request
	torrcContains(t, result, "TestSocks 1")

	// Strict node exclusion: if no nodes match EntryNodes/ExitNodes, fail
	// instead of falling back to unlisted nodes
	torrcContains(t, result, "StrictNodes 1")

	// Reject connections to internal/reserved IP addresses
	torrcContains(t, result, "ClientRejectInternalAddresses 1")

	// Exclude nodes with unknown GeoIP country (prevents nodes in unknown
	// jurisdictions from being selected)
	torrcContains(t, result, "GeoIPExcludeUnknown 1")

	// Warn about plaintext ports (DNS, SMTP, etc.)
	torrcContains(t, result, "WarnPlaintextPorts 21,23,25")

	// Automap .onion and .exit hostnames
	torrcContains(t, result, "AutomapHostsSuffixes .exit,.onion")

	// Cookie authentication for control port (never password auth)
	torrcContains(t, result, "CookieAuthentication 1")
	torrcNotContains(t, result, "HashedControlPassword")

	// Entry guards for long-term entry node selection
	torrcContains(t, result, "UseEntryGuards 1")
}

// TestTorrcTemplate_SecurityOptionsCannotBeDisabled tests that even when
// security options are set to their weakest allowed value, the hardcoded
// protections remain.
func TestTorrcTemplate_SecurityOptionsCannotBeDisabled(t *testing.T) {
	ic := baseInstanceConfig()
	// Set security options to 0 (disabled)
	ic.SafeSocks = 0
	ic.TestSocks = 0
	ic.StrictNodes = 0
	ic.ClientRejectInternalAddresses = 0
	ic.GeoIPExcludeUnknown = 0

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	// Configurable options now render as 0
	torrcContains(t, result, "SafeSocks 0")
	torrcContains(t, result, "StrictNodes 0")

	// But hardcoded options MUST still be present regardless of config
	torrcContains(t, result, "SafeLogging 1")
	torrcContains(t, result, "NoExec 1")
	torrcContains(t, result, "DisableDebuggerAttachment 1")
	torrcContains(t, result, "EnforceDistinctSubnets 1")
	torrcContains(t, result, "CookieAuthentication 1")
	torrcNotContains(t, result, "HashedControlPassword")
}

// TestTorrcTemplate_ControlPortSecurity verifies control port security:
// cookie auth is used, cookie file path is specified, no password auth.
func TestTorrcTemplate_ControlPortSecurity(t *testing.T) {
	ic := baseInstanceConfig()
	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	// Cookie authentication must be enabled
	torrcContains(t, result, "CookieAuthentication 1")

	// Cookie auth file must be in the instance data directory
	torrcContains(t, result, "CookieAuthFile /tmp/test/control_auth_cookie")

	// Password-based auth must NEVER appear
	torrcNotContains(t, result, "HashedControlPassword")
}

// TestTorrcTemplate_WarnPlaintextPorts renders correctly
func TestTorrcTemplate_WarnPlaintextPorts(t *testing.T) {
	ic := baseInstanceConfig()
	ic.WarnPlaintextPorts = "21,23,25,80,109,110,143"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "WarnPlaintextPorts 21,23,25,80,109,110,143")
}

// TestTorrcTemplate_AutomapHostsSuffixes renders correctly
func TestTorrcTemplate_AutomapHostsSuffixes(t *testing.T) {
	ic := baseInstanceConfig()
	ic.AutomapHostsSuffixes = ".exit,.onion"

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "AutomapHostsSuffixes .exit,.onion")
}

// TestTorrcTemplate_KeepalivePeriod renders correctly
func TestTorrcTemplate_KeepalivePeriod(t *testing.T) {
	ic := baseInstanceConfig()
	ic.KeepalivePeriod = 300

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "KeepalivePeriod 300")
}

// TestTorrcTemplate_LongLivedPorts renders correctly
func TestTorrcTemplate_LongLivedPorts(t *testing.T) {
	ic := baseInstanceConfig()
	ic.LongLivedPorts = []int{993, 995, 5222, 5223}

	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	torrcContains(t, result, "LongLivedPorts 993,995,5222,5223")
}

// TestTorrcTemplate_SecurityDirectivesLineCount verifies that the rendered
// torrc contains a minimum number of security-relevant directives. This is a
// regression test to catch accidental removal of security options.
func TestTorrcTemplate_SecurityDirectivesLineCount(t *testing.T) {
	ic := baseInstanceConfig()
	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	securityDirectives := []string{
		"SafeLogging 1",
		"NoExec 1",
		"DisableDebuggerAttachment 1",
		"EnforceDistinctSubnets 1",
		"SafeSocks 1",
		"StrictNodes 1",
		"ClientRejectInternalAddresses 1",
		"CookieAuthentication 1",
		"GeoIPExcludeUnknown 1",
		"ClientUseIPv4 1",
	}

	found := 0
	for _, directive := range securityDirectives {
		if strings.Contains(result, directive) {
			found++
		}
	}

	minExpected := len(securityDirectives)
	if found < minExpected {
		missing := []string{}
		for _, d := range securityDirectives {
			if !strings.Contains(result, d) {
				missing = append(missing, d)
			}
		}
		t.Errorf("only %d/%d security directives found, missing: %v", found, minExpected, missing)
	}
}

// TestTorrcTemplate_NoDeprecatedOptions verifies that known-deprecated or
// removed torrc options do NOT appear in the rendered output.
func TestTorrcTemplate_NoDeprecatedOptions(t *testing.T) {
	ic := baseInstanceConfig()
	result, err := RenderTorrc(ic, readTorrcTemplate(t))
	if err != nil {
		t.Fatalf("RenderTorrc() error = %v", err)
	}

	// These options were removed or made obsolete in modern Tor versions
	deprecated := []string{
		"CGOEnabled",                       // Not a torrc option (was erroneously emitted)
		"OptimisticData",                   // Obsolete in Tor 0.4.9.x
		"AllowDotExit",                     // Removed in Tor 0.4.x
		"ClientDNSRejectInternalAddresses", // Removed, replaced by ClientRejectInternalAddresses
	}

	for _, opt := range deprecated {
		if strings.Contains(result, opt) {
			t.Errorf("deprecated option %q found in rendered torrc\nfull output:\n%s", opt, result)
		}
	}
}
