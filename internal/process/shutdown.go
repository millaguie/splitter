package process

import (
	"context"
	"fmt"
	"sync"
	"syscall"
	"time"
)

const gracefulShutdownTimeout = 5 * time.Second

func (m *Manager) Stop(ctx context.Context, p *Process) error {
	if p == nil {
		return nil
	}

	p.mu.Lock()
	if p.state != StateRunning {
		p.mu.Unlock()
		<-p.done
		return nil
	}
	p.state = StateStopped
	p.mu.Unlock()

	if p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	pid := p.cmd.Process.Pid

	_ = syscall.Kill(-pid, syscall.SIGTERM)

	select {
	case <-p.done:
		return nil
	case <-time.After(gracefulShutdownTimeout):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	case <-ctx.Done():
		_ = syscall.Kill(-pid, syscall.SIGKILL)
	}

	<-p.done

	return nil
}

func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	processes := make([]*Process, len(m.processes))
	copy(processes, m.processes)
	m.mu.Unlock()

	var wg sync.WaitGroup
	errCh := make(chan error, len(processes))

	for _, p := range processes {
		wg.Add(1)
		go func(proc *Process) {
			defer wg.Done()
			if err := m.Stop(ctx, proc); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(p)
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("StopAll: %d process(es) failed: %w", len(errs), errs[0])
	}

	return nil
}
