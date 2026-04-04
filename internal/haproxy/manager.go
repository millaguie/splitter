package haproxy

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

type HAProxyManager struct {
	cfg           *config.Config
	procMgr       *process.Manager
	statsPassword string
	configFile    string
	proc          *process.Process
}

func NewManager(cfg *config.Config, procMgr *process.Manager) *HAProxyManager {
	return &HAProxyManager{
		cfg:           cfg,
		procMgr:       procMgr,
		statsPassword: generateStatsPassword(),
		configFile:    cfg.HAProxy.ConfigFile,
	}
}

func (m *HAProxyManager) GenerateConfig(torInstances []*tor.Instance) error {
	if err := os.MkdirAll(filepath.Dir(m.configFile), 0755); err != nil {
		return fmt.Errorf("GenerateConfig: mkdir: %w", err)
	}

	data := BuildConfigData(m.cfg, torInstances, m.statsPassword)

	tmpl, err := template.ParseFiles("templates/haproxy.cfg.gotmpl")
	if err != nil {
		return fmt.Errorf("GenerateConfig: parse template: %w", err)
	}

	f, err := os.Create(m.configFile)
	if err != nil {
		return fmt.Errorf("GenerateConfig: create %s: %w", m.configFile, err)
	}
	defer func() { _ = f.Close() }()

	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("GenerateConfig: execute: %w", err)
	}

	slog.Info("haproxy config generated", "path", m.configFile, "backends", len(data.HTTPBackends)+len(data.SOCKSBackends))
	return nil
}

func (m *HAProxyManager) Start(ctx context.Context) error {
	proc, err := m.procMgr.Spawn(ctx, "haproxy",
		m.cfg.HAProxy.BinaryPath,
		"-f", m.configFile,
	)
	if err != nil {
		return fmt.Errorf("Start: %w", err)
	}
	m.proc = proc

	slog.Info("haproxy started", "config", m.configFile)
	return nil
}

func (m *HAProxyManager) Stop(ctx context.Context) error {
	if m.proc == nil {
		return nil
	}
	if err := m.procMgr.Stop(ctx, m.proc); err != nil {
		return fmt.Errorf("Stop: %w", err)
	}
	m.proc = nil
	return nil
}

func (m *HAProxyManager) Reload(ctx context.Context) error {
	if m.proc == nil {
		return fmt.Errorf("Reload: haproxy not running")
	}

	pid := m.proc.Pid()
	if pid <= 0 {
		return fmt.Errorf("Reload: haproxy has no pid")
	}

	proc, err := m.procMgr.Spawn(ctx, "haproxy-reload",
		m.cfg.HAProxy.BinaryPath,
		"-f", m.configFile,
		"-sf", fmt.Sprintf("%d", pid),
	)
	if err != nil {
		return fmt.Errorf("Reload: %w", err)
	}

	m.proc = proc

	slog.Info("haproxy reloaded", "old_pid", pid)
	return nil
}

func (m *HAProxyManager) StatsPassword() string {
	return m.statsPassword
}

func RenderConfig(data *ConfigData, tmplStr string) (string, error) {
	tmpl, err := template.New("haproxy").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("RenderConfig: %w", err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("RenderConfig: execute: %w", err)
	}
	return buf.String(), nil
}
