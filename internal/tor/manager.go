package tor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/user/splitter/internal/cli"
	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

type TorManager struct {
	mu        sync.RWMutex
	instances []*Instance
	version   *Version
	procMgr   *process.Manager
	cfg       *config.Config
}

func NewManager(cfg *config.Config, procMgr *process.Manager) *TorManager {
	return &TorManager{
		cfg:     cfg,
		procMgr: procMgr,
	}
}

func (m *TorManager) DetectAndCreate(ctx context.Context, countries []string) error {
	v, err := DetectVersion(ctx, m.cfg.Tor.BinaryPath)
	if err != nil {
		return fmt.Errorf("DetectAndCreate: %w", err)
	}
	m.version = v

	slog.Info("detected tor version", "version", v.String(),
		"conflux", v.SupportsConflux(),
		"http_tunnel", v.SupportsHTTPTunnel(),
		"congestion_control", v.SupportsCongestionControl(),
		"cgo", v.SupportsCGO(),
		"post_quantum", v.SupportsPostQuantum(),
		"happy_families", v.SupportsHappyFamilies(),
		"tls13", v.SupportsTLS13(),
		"sandbox_supported", v.SupportsSandbox(),
	)

	perCountry := m.cfg.Instances.PerCountry
	socksPort := m.cfg.Tor.StartSocksPort
	controlPort := m.cfg.Tor.StartControlPort
	httpPort := m.cfg.Tor.StartHTTPPort

	instanceID := 0
	for _, country := range countries {
		for i := 0; i < perCountry; i++ {
			inst := NewInstance(instanceID, country, m.cfg, m.version, m.procMgr)
			inst.SocksPort = socksPort + instanceID
			inst.ControlPort = controlPort + instanceID
			if m.version.SupportsHTTPTunnel() {
				inst.HTTPPort = httpPort + instanceID
			}
			m.instances = append(m.instances, inst)
			instanceID++
		}
	}

	slog.Info("created tor instances", "count", len(m.instances), "countries", len(countries))
	return nil
}

func (m *TorManager) CreateFromVersion(version *Version, countries []string) {
	m.version = version

	perCountry := m.cfg.Instances.PerCountry
	socksPort := m.cfg.Tor.StartSocksPort
	controlPort := m.cfg.Tor.StartControlPort
	httpPort := m.cfg.Tor.StartHTTPPort

	instanceID := 0
	for _, country := range countries {
		for i := 0; i < perCountry; i++ {
			inst := NewInstance(instanceID, country, m.cfg, m.version, m.procMgr)
			inst.SocksPort = socksPort + instanceID
			inst.ControlPort = controlPort + instanceID
			if version.SupportsHTTPTunnel() {
				inst.HTTPPort = httpPort + instanceID
			}
			m.instances = append(m.instances, inst)
			instanceID++
		}
	}
}

func (m *TorManager) StartAll(ctx context.Context) error {
	m.mu.RLock()
	instances := make([]*Instance, len(m.instances))
	copy(instances, m.instances)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(instances))

	for _, inst := range instances {
		wg.Add(1)
		go func(i *Instance) {
			defer wg.Done()
			if err := i.Start(ctx); err != nil {
				errCh <- fmt.Errorf("instance %d (%s): %w", i.ID, i.Country, err)
			}
		}(inst)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("StartAll: %d instance(s) failed: %w", len(errs), errs[0])
	}

	slog.Info("all tor instances started", "count", len(instances))
	return nil
}

func (m *TorManager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	instances := make([]*Instance, len(m.instances))
	copy(instances, m.instances)
	m.mu.RUnlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(instances))

	for _, inst := range instances {
		wg.Add(1)
		go func(i *Instance) {
			defer wg.Done()
			if err := i.Stop(ctx); err != nil {
				errCh <- fmt.Errorf("instance %d: %w", i.ID, err)
			}
		}(inst)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("StopAll: %d instance(s) failed: %w", len(errs), errs[0])
	}

	return nil
}

func (m *TorManager) GetInstances() []*Instance {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Instance, len(m.instances))
	copy(out, m.instances)
	return out
}

func (m *TorManager) GetInstance(id int) (*Instance, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, inst := range m.instances {
		if inst.ID == id {
			return inst, nil
		}
	}
	return nil, fmt.Errorf("GetInstance: instance %d not found", id)
}

func (m *TorManager) GetVersion() *Version {
	return m.version
}

func (m *TorManager) StartAllWithRestart(ctx context.Context) error {
	m.mu.RLock()
	instances := make([]*Instance, len(m.instances))
	copy(instances, m.instances)
	m.mu.RUnlock()

	readyCh := make(chan struct{}, len(instances))

	for _, inst := range instances {
		go inst.RunWithRestart(ctx, readyCh)
	}

	startedCount := 0
	total := len(instances)
	for startedCount < total {
		select {
		case <-ctx.Done():
			return fmt.Errorf("StartAllWithRestart: %w", ctx.Err())
		case <-readyCh:
			startedCount++
			slog.Info("instance ready",
				cli.InstanceField(startedCount),
				"total", total,
			)
		}
	}

	slog.Info("all tor instances ready", "count", total)
	return nil
}

type InstanceInfo struct {
	ID      int
	Country string
}

func (m *TorManager) GetInstanceInfos() []InstanceInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]InstanceInfo, len(m.instances))
	for i, inst := range m.instances {
		infos[i] = InstanceInfo{
			ID:      inst.ID,
			Country: inst.Country,
		}
	}
	return infos
}

func (m *TorManager) RotateInstance(ctx context.Context, id int, newCountry string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, inst := range m.instances {
		if inst.ID == id {
			if err := inst.Stop(ctx); err != nil {
				return fmt.Errorf("RotateInstance: stop %d: %w", id, err)
			}
			inst.Country = newCountry
			if err := inst.Start(ctx); err != nil {
				return fmt.Errorf("RotateInstance: start %d: %w", id, err)
			}
			return nil
		}
	}
	return fmt.Errorf("RotateInstance: instance %d not found", id)
}

func (m *TorManager) SetBridgesForAll(bridgeType string, bridges *BridgesConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, inst := range m.instances {
		if err := inst.SetBridges(bridgeType, bridges); err != nil {
			return fmt.Errorf("SetBridgesForAll: instance %d: %w", inst.ID, err)
		}
	}
	return nil
}
