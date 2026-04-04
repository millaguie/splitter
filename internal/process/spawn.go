package process

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"syscall"
)

func (m *Manager) Spawn(ctx context.Context, name, binary string, args ...string) (*Process, error) {
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("Spawn: %w", ctx.Err())
	default:
	}

	cmd := exec.Command(binary, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	p := &Process{
		Name:  name,
		Path:  binary,
		Args:  args,
		cmd:   cmd,
		state: StateStarting,
		done:  make(chan struct{}),
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("Spawn: %w", err)
	}

	p.mu.Lock()
	p.state = StateRunning
	p.mu.Unlock()

	m.add(p)

	go func() {
		waitErr := cmd.Wait()
		p.mu.Lock()
		p.wait = waitErr
		if p.state == StateRunning {
			if waitErr != nil {
				p.state = StateFailed
			} else {
				p.state = StateStopped
			}
		}
		p.mu.Unlock()
		close(p.done)
	}()

	return p, nil
}
