package tor

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a parsed Tor version (major.minor.patch.release).
type Version struct {
	Major   int
	Minor   int
	Patch   int
	Release int
}

// DetectVersion runs `tor --version` and parses the version from the output.
func DetectVersion(ctx context.Context, binaryPath string) (*Version, error) {
	type result struct {
		v   *Version
		err error
	}
	ch := make(chan result, 1)

	go func() {
		output, err := detectVersionOutput(binaryPath)
		if err != nil {
			ch <- result{nil, err}
			return
		}
		v, err := parseVersionOutput(output)
		ch <- result{v, err}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("DetectVersion: %w", ctx.Err())
	case r := <-ch:
		return r.v, r.err
	}
}

func detectVersionOutput(binaryPath string) (string, error) {
	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("detectVersionOutput: %w", err)
	}
	return string(output), nil
}

// DetectVersionFromOutput parses a Tor version from a `tor --version` output string.
func DetectVersionFromOutput(output string) (*Version, error) {
	return parseVersionOutput(output)
}

// versionRegex matches "Tor version X.Y.Z" or "Tor version X.Y.Z.W"
var versionRegex = regexp.MustCompile(`Tor version (\d+)\.(\d+)\.(\d+)(?:\.(\d+))?`)

func parseVersionOutput(output string) (*Version, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := versionRegex.FindStringSubmatch(line)
		if len(matches) >= 4 {
			major, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			minor, err := strconv.Atoi(matches[2])
			if err != nil {
				continue
			}
			patch, err := strconv.Atoi(matches[3])
			if err != nil {
				continue
			}
			release := 0
			if len(matches) == 5 && matches[4] != "" {
				release, err = strconv.Atoi(matches[4])
				if err != nil {
					release = 0
				}
			}
			return &Version{Major: major, Minor: minor, Patch: patch, Release: release}, nil
		}
	}
	return nil, fmt.Errorf("parseVersionOutput: no Tor version found in %q", output)
}

func (v *Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Patch, v.Release)
}

// SupportsConflux returns true for Tor >= 0.4.8.
func (v *Version) SupportsConflux() bool {
	return v.atLeast(0, 4, 8, 0)
}

// SupportsHTTPTunnel returns true for Tor >= 0.4.8.
func (v *Version) SupportsHTTPTunnel() bool {
	return v.atLeast(0, 4, 8, 0)
}

// SupportsCongestionControl returns true for Tor >= 0.4.7.
func (v *Version) SupportsCongestionControl() bool {
	return v.atLeast(0, 4, 7, 0)
}

// SupportsCGO returns true for Tor >= 0.4.9 (Counter Galois Onion encryption).
func (v *Version) SupportsCGO() bool {
	return v.atLeast(0, 4, 9, 0)
}

// SupportsTLS13 returns true for Tor >= 0.4.9, which recommends TLS 1.3
// for link encryption. Actual TLS 1.3 support also depends on the linked
// OpenSSL version, but Tor 0.4.9+ is built to use it when available.
func (v *Version) SupportsTLS13() bool {
	return v.atLeast(0, 4, 9, 0)
}

// SupportsSandbox returns true for Tor >= 0.4.7.
// Tor's seccomp-bpf sandbox (Sandbox 1) has been available for a long time
// but had bugs in older versions. It is considered stable since 0.4.7+.
func (v *Version) SupportsSandbox() bool {
	return v.atLeast(0, 4, 7, 0)
}

// SupportsHappyFamilies returns true for Tor >= 0.4.9 (proposal 321).
// Happy Families groups relays from the same operator to avoid assigning
// multiple circuit hops to the same family. This is automatic in Tor 0.4.9+;
// no client-side torrc directives are needed.
func (v *Version) SupportsHappyFamilies() bool {
	return v.atLeast(0, 4, 9, 0)
}

// SupportsPostQuantum returns true for Tor >= 0.4.8.17, which introduced
// ML-KEM768 post-quantum key exchange when built with OpenSSL 3.5.0+.
// Note: actual PQ support also requires a compatible OpenSSL build; this
// only checks the Tor version prerequisite.
func (v *Version) SupportsPostQuantum() bool {
	return v.atLeast(0, 4, 8, 17)
}

func (v *Version) atLeast(major, minor, patch, release int) bool {
	if v.Major != major {
		return v.Major > major
	}
	if v.Minor != minor {
		return v.Minor > minor
	}
	if v.Patch != patch {
		return v.Patch > patch
	}
	return v.Release >= release
}

// TLSInfo holds detected TLS/OpenSSL capabilities from the Tor binary.
type TLSInfo struct {
	OpenSSLVersion string // e.g. "3.5.0", "1.1.1"
	PostQuantumOK  bool   // true if OpenSSL >= 3.5.0 (ML-KEM768 capable)
}

// DetectTLSInfo parses OpenSSL version from the full `tor --version` output.
// The second line typically contains: "Tor is running on Linux with ... OpenSSL X.Y.Z ..."
func DetectTLSInfo(ctx context.Context, binaryPath string) (*TLSInfo, error) {
	type result struct {
		info *TLSInfo
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		output, err := detectVersionOutput(binaryPath)
		if err != nil {
			ch <- result{nil, err}
			return
		}
		info := parseTLSInfo(output)
		ch <- result{info, nil}
	}()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("DetectTLSInfo: %w", ctx.Err())
	case r := <-ch:
		return r.info, r.err
	}
}

// DetectTLSInfoFromOutput parses TLS info from a `tor --version` output string.
func DetectTLSInfoFromOutput(output string) *TLSInfo {
	return parseTLSInfo(output)
}

var openSSLRegex = regexp.MustCompile(`OpenSSL\s+(\d+)\.(\d+)\.(\d+)`)

func parseTLSInfo(output string) *TLSInfo {
	info := &TLSInfo{}

	matches := openSSLRegex.FindStringSubmatch(output)
	if len(matches) == 4 {
		info.OpenSSLVersion = matches[0] // "OpenSSL X.Y.Z"

		major, err1 := strconv.Atoi(matches[1])
		minor, err2 := strconv.Atoi(matches[2])
		patch, err3 := strconv.Atoi(matches[3])

		if err1 == nil && err2 == nil && err3 == nil {
			info.PostQuantumOK = isOpenSSLAtLeast(major, minor, patch, 3, 5, 0)
		}
	}

	return info
}

func isOpenSSLAtLeast(major, minor, patch, wantMajor, wantMinor, wantPatch int) bool {
	if major != wantMajor {
		return major > wantMajor
	}
	if minor != wantMinor {
		return minor > wantMinor
	}
	return patch >= wantPatch
}
