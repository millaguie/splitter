package health

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/tor"
)

type CheckResult struct {
	TorPath     string
	TorVersion  *tor.Version
	HAProxyPath string
	PrivoxyPath string
	Features    FeatureSupport
}

type FeatureSupport struct {
	Conflux           bool
	HTTPTunnel        bool
	CongestionControl bool
	CGO               bool
	PostQuantum       bool
}

var lookPath = exec.LookPath

var runVersionCmd = func(ctx context.Context, binary string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, "--version")
	return cmd.Output()
}

func CheckDependencies(ctx context.Context, cfg *config.Config) (*CheckResult, error) {
	torPath, err := lookPath(cfg.Tor.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("dependency check: tor binary not found: %s is not in PATH or not executable", cfg.Tor.BinaryPath)
	}

	haProxyPath, err := lookPath(cfg.HAProxy.BinaryPath)
	if err != nil {
		return nil, fmt.Errorf("dependency check: haproxy binary not found: %s is not in PATH or not executable", cfg.HAProxy.BinaryPath)
	}

	var privoxyPath string
	if cfg.ProxyMode == "legacy" {
		p, err := lookPath(cfg.Privoxy.BinaryPath)
		if err != nil {
			return nil, fmt.Errorf("dependency check: privoxy binary not found (required for legacy proxy mode): %s is not in PATH or not executable", cfg.Privoxy.BinaryPath)
		}
		privoxyPath = p
	}

	output, err := runVersionCmd(ctx, torPath)
	if err != nil {
		return nil, fmt.Errorf("CheckDependencies: failed to detect tor version: %w", err)
	}

	version, err := tor.DetectVersionFromOutput(string(output))
	if err != nil {
		return nil, fmt.Errorf("CheckDependencies: failed to parse tor version: %w", err)
	}

	features := FeatureSupport{
		Conflux:           version.SupportsConflux(),
		HTTPTunnel:        version.SupportsHTTPTunnel(),
		CongestionControl: version.SupportsCongestionControl(),
		CGO:               version.SupportsCGO(),
		PostQuantum:       version.SupportsPostQuantum(),
	}

	slog.Info("dependency check passed",
		"tor_path", torPath,
		"tor_version", version.String(),
		"haproxy_path", haProxyPath,
		"privoxy_path", privoxyPath,
		"conflux", features.Conflux,
		"http_tunnel", features.HTTPTunnel,
		"congestion_control", features.CongestionControl,
		"cgo", features.CGO,
	)

	return &CheckResult{
		TorPath:     torPath,
		TorVersion:  version,
		HAProxyPath: haProxyPath,
		PrivoxyPath: privoxyPath,
		Features:    features,
	}, nil
}
