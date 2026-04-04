package process

import (
	"fmt"
	"os/exec"
	"sync"
)

type ProcessState int

const (
	StateStarting ProcessState = iota
	StateRunning
	StateStopped
	StateFailed
)

func (s ProcessState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopped:
		return "stopped"
	case StateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

type Process struct {
	Name string
	Path string
	Args []string

	mu    sync.Mutex
	cmd   *exec.Cmd
	state ProcessState
	done  chan struct{}
	wait  error
}

func (p *Process) State() ProcessState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Process) Pid() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Pid
	}
	return 0
}

func (p *Process) Wait() error {
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.wait
}

type Manager struct {
	mu        sync.Mutex
	processes []*Process
	tmpDir    string
}

func NewManager(tmpDir string) *Manager {
	return &Manager{
		tmpDir: tmpDir,
	}
}

func (m *Manager) List() []*Process {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Process, len(m.processes))
	copy(out, m.processes)
	return out
}

func (m *Manager) add(p *Process) {
	m.mu.Lock()
	m.processes = append(m.processes, p)
	m.mu.Unlock()
}

func (m *Manager) Wait(p *Process) error {
	if p == nil {
		return fmt.Errorf("Wait: nil process")
	}
	return p.Wait()
}
