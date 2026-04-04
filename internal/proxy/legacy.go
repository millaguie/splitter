package proxy

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"text/template"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

type LegacyProxy struct {
	cfg     *config.Config
	procMgr *process.Manager
	ports   []int
	procs   []*process.Process
}

type privoxyConfigData struct {
	InstanceID int
	ListenAddr string
	Port       int
	SocksPort  int
}

const defaultPrivoxyTmpl = `# SPLITTER Privoxy Config - Instance {{.InstanceID}}
# Generated automatically - do not edit

listen-address {{.ListenAddr}}:{{.Port}}
forward-socks5t / 127.0.0.1:{{.SocksPort}} .
forward         168.192.0.0/16 .
forward         10.0.0.0/8 .
forward         172.16.0.0/12 .
forward         192.168.0.0/16 .
forward         127.0.0.0/8 .
forward         0.0.0.0/8 .
forward         169.254.0.0/16 .

# Security
toggle  1
enable-remote-toggle 0
enable-edit-actions 0
enforce-blocks 1

# Logging (off by default)
logfile /dev/null

# Misc
buffer-limit 4096
`

func (p *LegacyProxy) Setup(_ context.Context, instances []Instance) ([]int, error) {
	tmpl, err := template.New("privoxy").Parse(defaultPrivoxyTmpl)
	if err != nil {
		return nil, fmt.Errorf("Setup: parse template: %w", err)
	}

	p.ports = make([]int, len(instances))

	for i, inst := range instances {
		privoxyPort := p.cfg.Privoxy.StartPort + inst.ID
		p.ports[i] = privoxyPort

		data := privoxyConfigData{
			InstanceID: inst.ID,
			ListenAddr: p.cfg.Privoxy.Listen,
			Port:       privoxyPort,
			SocksPort:  inst.SocksPort,
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("Setup: render config for instance %d: %w", inst.ID, err)
		}

		configPath := fmt.Sprintf("%s%d.cfg", p.cfg.Privoxy.ConfigFilePrefix, inst.ID)
		if err := os.WriteFile(configPath, buf.Bytes(), 0600); err != nil {
			return nil, fmt.Errorf("Setup: write config for instance %d: %w", inst.ID, err)
		}
	}

	return p.ports, nil
}

func (p *LegacyProxy) Start(ctx context.Context) error {
	if len(p.ports) == 0 {
		return nil
	}

	p.procs = make([]*process.Process, len(p.ports))

	for i := range p.ports {
		configPath := fmt.Sprintf("%s%d.cfg", p.cfg.Privoxy.ConfigFilePrefix, i)
		name := fmt.Sprintf("privoxy-%d", i)

		proc, err := p.procMgr.Spawn(ctx, name, p.cfg.Privoxy.BinaryPath, configPath)
		if err != nil {
			for j := 0; j < i; j++ {
				_ = p.procMgr.Stop(ctx, p.procs[j])
			}
			return fmt.Errorf("Start: spawn privoxy instance %d: %w", i, err)
		}
		p.procs[i] = proc
	}

	return nil
}

func (p *LegacyProxy) Stop(ctx context.Context) error {
	var firstErr error
	for i, proc := range p.procs {
		if proc == nil {
			continue
		}
		if err := p.procMgr.Stop(ctx, proc); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("Stop: privoxy instance %d: %w", i, err)
		}
	}
	p.procs = nil
	return firstErr
}

func (p *LegacyProxy) Mode() Mode {
	return ModeLegacy
}

func RenderPrivoxyConfig(data privoxyConfigData, tmplStr string) (string, error) {
	tmpl, err := template.New("privoxy").Parse(tmplStr)
	if err != nil {
		return "", fmt.Errorf("RenderPrivoxyConfig: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("RenderPrivoxyConfig: execute: %w", err)
	}
	return buf.String(), nil
}
