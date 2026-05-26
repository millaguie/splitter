package health

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/user/splitter/internal/config"
)

func TestCheckDependencies_allFound(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) {
		return name, nil
	}
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return []byte("Tor version 0.4.8.10.\n"), nil
	}

	cfg := testConfig("native")
	result, err := CheckDependencies(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CheckDependencies() error = %v", err)
	}
	if result.TorPath != cfg.Tor.BinaryPath {
		t.Errorf("TorPath = %q, want %q", result.TorPath, cfg.Tor.BinaryPath)
	}
	if result.HAProxyPath != cfg.HAProxy.BinaryPath {
		t.Errorf("HAProxyPath = %q, want %q", result.HAProxyPath, cfg.HAProxy.BinaryPath)
	}
	if result.TorVersion.String() != "0.4.8.10" {
		t.Errorf("TorVersion = %q, want %q", result.TorVersion.String(), "0.4.8.10")
	}
	if result.PrivoxyPath != "" {
		t.Errorf("PrivoxyPath = %q, want empty in native mode", result.PrivoxyPath)
	}
}

func TestCheckDependencies_missingTor(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "/usr/bin/tor" {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return name, nil
	}

	cfg := testConfig("native")
	_, err := CheckDependencies(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
	if want := "dependency check: tor binary not found"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestCheckDependencies_missingHAProxy(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "/usr/sbin/haproxy" {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return name, nil
	}

	cfg := testConfig("native")
	_, err := CheckDependencies(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
	if want := "dependency check: haproxy binary not found"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestCheckDependencies_missingPrivoxyLegacyMode(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(name string) (string, error) {
		if name == "/usr/sbin/privoxy" {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return name, nil
	}

	cfg := testConfig("legacy")
	_, err := CheckDependencies(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
	if want := "dependency check: privoxy binary not found (required for legacy proxy mode)"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestCheckDependencies_nativeModeNoPrivoxy(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) {
		if name == "/usr/sbin/privoxy" {
			return "", &exec.Error{Name: name, Err: exec.ErrNotFound}
		}
		return name, nil
	}
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return []byte("Tor version 0.4.8.10.\n"), nil
	}

	cfg := testConfig("native")
	result, err := CheckDependencies(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CheckDependencies() error = %v", err)
	}
	if result.PrivoxyPath != "" {
		t.Errorf("PrivoxyPath = %q, want empty in native mode", result.PrivoxyPath)
	}
}

func TestCheckDependencies_privoxyFoundLegacyMode(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) {
		return name, nil
	}
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return []byte("Tor version 0.4.8.10.\n"), nil
	}

	cfg := testConfig("legacy")
	result, err := CheckDependencies(context.Background(), cfg)
	if err != nil {
		t.Fatalf("CheckDependencies() error = %v", err)
	}
	if result.PrivoxyPath != cfg.Privoxy.BinaryPath {
		t.Errorf("PrivoxyPath = %q, want %q", result.PrivoxyPath, cfg.Privoxy.BinaryPath)
	}
}

func TestCheckDependencies_featureDetection(t *testing.T) {
	tests := []struct {
		name           string
		versionOutput  string
		wantConflux    bool
		wantHTTPTunnel bool
		wantCongestion bool
		wantCGO        bool
	}{
		{
			name:           "0.4.7 - congestion only",
			versionOutput:  "Tor version 0.4.7.0.\n",
			wantConflux:    false,
			wantHTTPTunnel: false,
			wantCongestion: true,
			wantCGO:        false,
		},
		{
			name:           "0.4.8 - conflux and tunnel",
			versionOutput:  "Tor version 0.4.8.10.\n",
			wantConflux:    true,
			wantHTTPTunnel: true,
			wantCongestion: true,
			wantCGO:        false,
		},
		{
			name:           "0.4.9 - all features",
			versionOutput:  "Tor version 0.4.9.5.\n",
			wantConflux:    true,
			wantHTTPTunnel: true,
			wantCongestion: true,
			wantCGO:        true,
		},
		{
			name:           "0.4.6 - no features",
			versionOutput:  "Tor version 0.4.6.99.\n",
			wantConflux:    false,
			wantHTTPTunnel: false,
			wantCongestion: false,
			wantCGO:        false,
		},
	}

	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) { return name, nil }

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
				return []byte(tt.versionOutput), nil
			}

			cfg := testConfig("native")
			result, err := CheckDependencies(context.Background(), cfg)
			if err != nil {
				t.Fatalf("CheckDependencies() error = %v", err)
			}
			if result.Features.Conflux != tt.wantConflux {
				t.Errorf("Conflux = %v, want %v", result.Features.Conflux, tt.wantConflux)
			}
			if result.Features.HTTPTunnel != tt.wantHTTPTunnel {
				t.Errorf("HTTPTunnel = %v, want %v", result.Features.HTTPTunnel, tt.wantHTTPTunnel)
			}
			if result.Features.CongestionControl != tt.wantCongestion {
				t.Errorf("CongestionControl = %v, want %v", result.Features.CongestionControl, tt.wantCongestion)
			}
			if result.Features.CGO != tt.wantCGO {
				t.Errorf("CGO = %v, want %v", result.Features.CGO, tt.wantCGO)
			}
		})
	}
}

func TestCheckDependencies_versionCommandFails(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) { return name, nil }
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return nil, errors.New("exit status 1")
	}

	cfg := testConfig("native")
	_, err := CheckDependencies(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
	if want := "failed to detect tor version"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestCheckDependencies_unparseableVersion(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) { return name, nil }
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return []byte("some random output without version\n"), nil
	}

	cfg := testConfig("native")
	_, err := CheckDependencies(context.Background(), cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
	if want := "failed to parse tor version"; !contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err.Error(), want)
	}
}

func TestCheckDependencies_contextCancelled(t *testing.T) {
	origLookPath := lookPath
	origRunVersion := runVersionCmd
	defer func() {
		lookPath = origLookPath
		runVersionCmd = origRunVersion
	}()

	lookPath = func(name string) (string, error) { return name, nil }
	runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
		return nil, ctx.Err()
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	cfg := testConfig("native")
	_, err := CheckDependencies(ctx, cfg)
	if err == nil {
		t.Fatal("CheckDependencies() expected error, got nil")
	}
}

func testConfig(proxyMode string) *config.Config {
	cfg := &config.Config{}
	cfg.Tor.BinaryPath = "/usr/bin/tor"
	cfg.HAProxy.BinaryPath = "/usr/sbin/haproxy"
	cfg.Privoxy.BinaryPath = "/usr/sbin/privoxy"
	cfg.ProxyMode = proxyMode
	return cfg
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstr(s, substr))
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
