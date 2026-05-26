package tor

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/user/splitter/internal/cli"
)

const (
	maxBackoff     = 30 * time.Second
	initialBackoff = 1 * time.Second
	bootstrapGrace = 10 * time.Second
)

func (inst *Instance) RunWithRestart(ctx context.Context, readyCh chan<- struct{}) {
	var (
		failures  int
		onceReady sync.Once
	)

	for {
		instCtx, cancel := context.WithCancel(ctx)
		inst.cancelFunc = cancel

		inst.setState(StateBootstrapping)
		err := inst.Start(instCtx)
		if err != nil {
			slog.Error("tor instance start failed",
				cli.InstanceField(inst.ID),
				"error", err,
			)
			inst.setState(StateFailed)

			failures++
			backoff := backoffDuration(failures)
			slog.Warn("restarting tor instance after backoff",
				cli.InstanceField(inst.ID),
				"backoff", backoff,
				"failures", failures,
			)

			select {
			case <-ctx.Done():
				cancel()
				return
			case <-time.After(backoff):
				cancel()
				continue
			}
		}

		waitCh := make(chan error, 1)
		go func() {
			waitCh <- inst.Wait()
		}()

		bootstrapTimer := time.NewTimer(bootstrapGrace)

		restart := false
		for {
			select {
			case <-ctx.Done():
				bootstrapTimer.Stop()
				cancel()
				return
			case <-bootstrapTimer.C:
				if inst.GetState() == StateBootstrapping {
					inst.setState(StateReady)
					failures = 0
					onceReady.Do(func() {
						select {
						case readyCh <- struct{}{}:
						default:
						}
					})
				}
			case waitErr := <-waitCh:
				bootstrapTimer.Stop()
				if waitErr != nil && inst.GetState() != StateFailed {
					slog.Error("tor instance exited unexpectedly",
						cli.InstanceField(inst.ID),
						cli.CountryField(inst.Country),
						"error", waitErr,
					)
					inst.setState(StateFailed)

					failures++
					backoff := backoffDuration(failures)
					slog.Warn("restarting tor instance",
						cli.InstanceField(inst.ID),
						"backoff", backoff,
						"failures", failures,
					)

					select {
					case <-ctx.Done():
						cancel()
						return
					case <-time.After(backoff):
						cancel()
						restart = true
					}
				} else {
					cancel()
					return
				}
			}
			if restart {
				break
			}
		}
	}
}

func backoffDuration(failures int) time.Duration {
	if failures <= 0 {
		return initialBackoff
	}
	d := initialBackoff
	for i := 1; i < failures; i++ {
		d *= 2
		if d >= maxBackoff {
			return maxBackoff
		}
	}
	if d > maxBackoff {
		return maxBackoff
	}
	return d
}
