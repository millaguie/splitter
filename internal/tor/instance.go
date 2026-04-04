package tor

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"github.com/user/splitter/internal/cli"
	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

type State int

const (
	StateStarting State = iota
	StateBootstrapping
	StateReady
	StateFailed
)

func (s State) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateBootstrapping:
		return "bootstrapping"
	case StateReady:
		return "ready"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type Instance struct {
	ID          int
	Country     string
	SocksPort   int
	ControlPort int
	HTTPPort    int

	mu         sync.RWMutex
	state      State
	cancelFunc context.CancelFunc
	proc       *process.Process

	cfg     *config.Config
	version *Version
	procMgr *process.Manager

	torrcPath       string
	dataDir         string
	bridgeLines     []string
	bridgeTransport string
}

type InstanceConfig struct {
	InstanceID                    int
	Country                       string
	SocksPort                     int
	ControlPort                   int
	HTTPTunnelPort                int
	DataDir                       string
	CircuitBuildTimeout           int
	CircuitStreamTimeout          int
	MaxCircuitDirtiness           int
	NewCircuitPeriod              int
	LearnCircuitBuildTimeout      int
	CongestionControlAuto         bool
	ConfluxEnabled                bool
	PostQuantumAvailable          bool
	HappyFamiliesAware            bool
	TLS13Recommended              bool
	SandboxEnabled                bool
	RelayEnforce                  string
	HiddenServiceEnabled          bool
	HiddenServiceDir              string
	HiddenServicePort             int
	ConnectionPadding             int
	ReducedConnectionPadding      int
	SafeSocks                     int
	TestSocks                     int
	ClientRejectInternalAddresses int
	StrictNodes                   int
	ClientOnly                    int
	GeoIPExcludeUnknown           int
	FascistFirewall               int
	FirewallPorts                 []int
	LongLivedPorts                []int
	MaxClientCircuitsPending      int
	SocksTimeout                  int
	TrackHostExitsExpire          int
	UseEntryGuards                int
	NumEntryGuards                int
	AutomapHostsSuffixes          string
	WarnPlaintextPorts            string
	RejectPlaintextPorts          string
	KeepalivePeriod               int
	ControlAuth                   string
	BridgeLines                   []string
	UseBridges                    bool
	ClientTransport               string
	StreamIsolation               bool
	ClientUseIPv6                 bool
}

func NewInstance(id int, country string, cfg *config.Config, version *Version, procMgr *process.Manager) *Instance {
	return &Instance{
		ID:      id,
		Country: country,
		cfg:     cfg,
		version: version,
		procMgr: procMgr,
		state:   StateStarting,
	}
}

func (inst *Instance) SetBridges(bridgeType string, bridges *BridgesConfig) error {
	if bridgeType == "none" || bridgeType == "" {
		return nil
	}
	if bridges == nil {
		return fmt.Errorf("SetBridges: bridge config is nil")
	}
	bc, err := bridges.GetBridge(bridgeType)
	if err != nil {
		return fmt.Errorf("SetBridges: %w", err)
	}
	if bc != nil {
		inst.bridgeLines = bc.Lines
		inst.bridgeTransport = bc.Transport
	}
	return nil
}

func (inst *Instance) GetState() State {
	inst.mu.RLock()
	defer inst.mu.RUnlock()
	return inst.state
}

func (inst *Instance) setState(s State) {
	inst.mu.Lock()
	inst.state = s
	inst.mu.Unlock()
}

func (inst *Instance) Start(ctx context.Context) error {
	torrcPath, err := inst.generateTorrc()
	if err != nil {
		return fmt.Errorf("Start: %w", err)
	}
	inst.torrcPath = torrcPath

	dataDir, err := inst.generateDataDir()
	if err != nil {
		return fmt.Errorf("Start: %w", err)
	}
	inst.dataDir = dataDir

	inst.setState(StateBootstrapping)

	proc, err := inst.procMgr.Spawn(ctx, inst.processName(), inst.cfg.Tor.BinaryPath, "-f", torrcPath)
	if err != nil {
		inst.setState(StateFailed)
		return fmt.Errorf("Start: %w", err)
	}
	inst.proc = proc

	slog.Info("tor instance started",
		cli.InstanceField(inst.ID),
		cli.CountryField(inst.Country),
		cli.PortField(inst.SocksPort),
	)

	return nil
}

func (inst *Instance) Stop(ctx context.Context) error {
	if inst.cancelFunc != nil {
		inst.cancelFunc()
	}
	if inst.proc != nil {
		if err := inst.procMgr.Stop(ctx, inst.proc); err != nil {
			return fmt.Errorf("Stop: %w", err)
		}
	}
	inst.setState(StateStarting)
	return nil
}

func (inst *Instance) Wait() error {
	if inst.proc == nil {
		return nil
	}
	return inst.proc.Wait()
}

func (inst *Instance) processName() string {
	return fmt.Sprintf("tor-%d", inst.ID)
}

func (inst *Instance) generateTorrc() (string, error) {
	ic := inst.buildInstanceConfig()

	tmpl, err := template.ParseFiles("templates/torrc.gotmpl")
	if err != nil {
		return "", fmt.Errorf("generateTorrc: %w", err)
	}

	torrcPath := filepath.Join(inst.cfg.Paths.TempFiles, fmt.Sprintf("tor_%d.cfg", inst.ID))
	f, err := os.Create(torrcPath)
	if err != nil {
		return "", fmt.Errorf("generateTorrc: create %s: %w", torrcPath, err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, ic); err != nil {
		return "", fmt.Errorf("generateTorrc: execute: %w", err)
	}

	return torrcPath, nil
}

func (inst *Instance) generateDataDir() (string, error) {
	dataDir := filepath.Join(inst.cfg.Paths.TempFiles, fmt.Sprintf("tor_data_%d", inst.ID))
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", fmt.Errorf("generateDataDir: mkdir %s: %w", dataDir, err)
	}
	return dataDir, nil
}

func (inst *Instance) buildInstanceConfig() InstanceConfig {
	ic := InstanceConfig{
		InstanceID:                    inst.ID,
		Country:                       inst.Country,
		SocksPort:                     inst.SocksPort,
		ControlPort:                   inst.ControlPort,
		HTTPTunnelPort:                inst.HTTPPort,
		DataDir:                       filepath.Join(inst.cfg.Paths.TempFiles, fmt.Sprintf("tor_data_%d", inst.ID)),
		CircuitBuildTimeout:           inst.cfg.Tor.CircuitBuildTimeout,
		CircuitStreamTimeout:          inst.cfg.Tor.CircuitStreamTimeout,
		MaxCircuitDirtiness:           inst.randomizeDirtiness(),
		NewCircuitPeriod:              inst.cfg.Tor.NewCircuitPeriod,
		LearnCircuitBuildTimeout:      inst.cfg.Tor.LearnCircuitBuildTimeout,
		CongestionControlAuto:         inst.version.SupportsCongestionControl() && inst.cfg.Tor.CongestionControlAuto,
		ConfluxEnabled:                inst.version.SupportsConflux() && inst.cfg.Tor.ConfluxEnabled,
		PostQuantumAvailable:          inst.version.SupportsPostQuantum(),
		HappyFamiliesAware:            inst.version.SupportsHappyFamilies(),
		TLS13Recommended:              inst.version.SupportsTLS13(),
		SandboxEnabled:                inst.version.SupportsSandbox() && inst.cfg.Tor.Sandbox,
		RelayEnforce:                  inst.cfg.Relay.Enforce,
		HiddenServiceEnabled:          inst.cfg.Tor.HiddenService.Enabled,
		HiddenServiceDir:              inst.cfg.Tor.HiddenService.BasePath + fmt.Sprintf("%d", inst.ID),
		HiddenServicePort:             inst.cfg.Tor.HiddenService.StartPort + inst.ID,
		ConnectionPadding:             inst.cfg.Tor.ConnectionPadding,
		ReducedConnectionPadding:      inst.cfg.Tor.ReducedConnectionPadding,
		SafeSocks:                     inst.cfg.Tor.SafeSocks,
		TestSocks:                     inst.cfg.Tor.TestSocks,
		ClientRejectInternalAddresses: inst.cfg.Tor.ClientRejectInternalAddresses,
		StrictNodes:                   inst.cfg.Tor.StrictNodes,
		ClientOnly:                    inst.cfg.Tor.ClientOnly,
		GeoIPExcludeUnknown:           inst.cfg.Tor.GeoIPExcludeUnknown,
		FascistFirewall:               inst.cfg.Tor.FascistFirewall,
		FirewallPorts:                 inst.cfg.Tor.FirewallPorts,
		LongLivedPorts:                inst.cfg.Tor.LongLivedPorts,
		MaxClientCircuitsPending:      inst.cfg.Tor.MaxClientCircuitsPending,
		SocksTimeout:                  inst.cfg.Tor.SocksTimeout,
		TrackHostExitsExpire:          inst.cfg.Tor.TrackHostExitsExpire,
		UseEntryGuards:                inst.cfg.Tor.UseEntryGuards,
		NumEntryGuards:                inst.cfg.Tor.NumEntryGuards,
		AutomapHostsSuffixes:          inst.cfg.Tor.AutomapHostsSuffixes,
		WarnPlaintextPorts:            inst.cfg.Tor.WarnPlaintextPorts,
		RejectPlaintextPorts:          inst.cfg.Tor.RejectPlaintextPorts,
		KeepalivePeriod:               inst.cfg.Tor.MinimumTimeout,
		ControlAuth:                   inst.cfg.Tor.ControlAuth,
		BridgeLines:                   inst.bridgeLines,
		UseBridges:                    len(inst.bridgeLines) > 0,
		ClientTransport:               inst.bridgeTransport,
		StreamIsolation:               inst.cfg.Tor.StreamIsolation,
		ClientUseIPv6:                 inst.cfg.Tor.IPv6,
	}

	if inst.HTTPPort > 0 && inst.version.SupportsHTTPTunnel() {
		ic.HTTPTunnelPort = inst.HTTPPort
	}

	return ic
}

func (inst *Instance) randomizeDirtiness() int {
	max := inst.cfg.Tor.MaxCircuitDirtiness
	if max <= 10 {
		return max
	}
	return 10 + rand.Intn(max-10)
}

func RenderTorrc(ic InstanceConfig, tmplStr string) (string, error) {
	tmpl, err := template.New("torrc").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("RenderTorrc: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, ic); err != nil {
		return "", fmt.Errorf("RenderTorrc: execute: %w", err)
	}
	return buf.String(), nil
}
