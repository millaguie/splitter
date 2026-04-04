package circuit

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"time"

	"github.com/user/splitter/internal/cli"
)

const (
	defaultMinInterval = 10 * time.Second
	defaultMaxInterval = 15 * time.Second
)

type circuitInstance struct {
	id          int
	controlAddr string
	cookiePath  string
	minInterval time.Duration
	maxInterval time.Duration
}

type Renewer struct {
	instances  []*circuitInstance
	cancelFunc context.CancelFunc
	done       chan struct{}
	started    bool
	pattern    *TrafficPattern
}

func NewRenewer() *Renewer {
	return &Renewer{}
}

func (r *Renewer) AddInstance(id int, controlPort int, cookiePath string) {
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", controlPort))
	r.instances = append(r.instances, &circuitInstance{
		id:          id,
		controlAddr: addr,
		cookiePath:  cookiePath,
		minInterval: defaultMinInterval,
		maxInterval: defaultMaxInterval,
	})
}

func (r *Renewer) SetTrafficPattern(tp *TrafficPattern) {
	r.pattern = tp
}

func (r *Renewer) Start(ctx context.Context) error {
	if len(r.instances) == 0 {
		return fmt.Errorf("Start: no instances added")
	}

	ctx, cancel := context.WithCancel(ctx)
	r.cancelFunc = cancel
	r.done = make(chan struct{})
	r.started = true

	go func() {
		defer close(r.done)
		r.runAll(ctx)
	}()

	slog.Info("circuit renewal started", "instances", len(r.instances))
	return nil
}

func (r *Renewer) Stop() error {
	if !r.started {
		return nil
	}
	if r.cancelFunc != nil {
		r.cancelFunc()
	}
	<-r.done
	slog.Info("circuit renewal stopped")
	return nil
}

func (r *Renewer) runAll(ctx context.Context) {
	doneCh := make(chan struct{}, len(r.instances))

	for _, inst := range r.instances {
		go func(ci *circuitInstance) {
			r.runInstance(ctx, ci)
			doneCh <- struct{}{}
		}(inst)
	}

	for range r.instances {
		select {
		case <-ctx.Done():
			return
		case <-doneCh:
		}
	}
}

func (r *Renewer) runInstance(ctx context.Context, ci *circuitInstance) {
	client := NewClient(ci.controlAddr, ci.cookiePath)

	if err := client.Connect(ctx); err != nil {
		slog.Error("circuit renewal connect failed",
			cli.InstanceField(ci.id),
			"error", err,
		)
		return
	}
	defer func() { _ = client.Close() }()

	if err := client.Authenticate(ctx); err != nil {
		slog.Error("circuit renewal auth failed",
			cli.InstanceField(ci.id),
			"error", err,
		)
		return
	}

	slog.Info("circuit renewal connected",
		cli.InstanceField(ci.id),
		"addr", ci.controlAddr,
	)

	for {
		var interval time.Duration
		var mode string
		if r.pattern != nil {
			interval, mode = AdaptiveInterval(r.pattern, ci.minInterval, ci.maxInterval)
		} else {
			interval = randomInterval(ci.minInterval, ci.maxInterval)
			mode = "fixed"
		}
		t := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}

		if err := client.SignalNewnym(ctx); err != nil {
			slog.Error("circuit renewal NEWNYM failed, reconnecting",
				cli.InstanceField(ci.id),
				"error", err,
			)

			_ = client.Close()

			if err := r.reconnect(ctx, client); err != nil {
				slog.Error("circuit renewal reconnect failed",
					cli.InstanceField(ci.id),
					"error", err,
				)
				return
			}
			continue
		}

		slog.Debug("circuit renewed",
			cli.InstanceField(ci.id),
			"interval", interval,
			"mode", mode,
		)
	}
}

func (r *Renewer) reconnect(ctx context.Context, client *Client) error {
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("reconnect: connect: %w", err)
	}
	if err := client.Authenticate(ctx); err != nil {
		return fmt.Errorf("reconnect: auth: %w", err)
	}
	return nil
}

func randomInterval(min, max time.Duration) time.Duration {
	delta := max - min
	if delta <= 0 {
		return min
	}
	return min + time.Duration(rand.Int64N(int64(delta)))
}
