package country

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/user/splitter/internal/config"
)

type InstanceInfo struct {
	ID      int
	Country string
}

type InstanceRotator interface {
	GetInstances() []InstanceInfo
	RotateInstance(ctx context.Context, id int, newCountry string) error
}

const maxJitter = 30 * time.Second

type Daemon struct {
	mu           sync.Mutex
	cfg          *config.Config
	rotator      InstanceRotator
	baseInterval time.Duration
	cancelFunc   context.CancelFunc
	done         chan struct{}
	started      bool
}

func NewDaemon(cfg *config.Config, rotator InstanceRotator) *Daemon {
	return &Daemon{
		cfg:          cfg,
		rotator:      rotator,
		baseInterval: time.Duration(cfg.Country.Rotation.Interval) * time.Second,
	}
}

func (d *Daemon) Start(ctx context.Context) error {
	d.done = make(chan struct{})
	d.started = true

	if !d.cfg.Country.Rotation.Enabled {
		slog.Info("country rotation disabled")
		close(d.done)
		return nil
	}

	ctx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel

	go d.run(ctx)

	slog.Info("country rotation daemon started", "base_interval", d.baseInterval)
	return nil
}

func (d *Daemon) Stop() error {
	if !d.started {
		return nil
	}
	if d.cancelFunc != nil {
		d.cancelFunc()
	}
	<-d.done
	return nil
}

func (d *Daemon) run(ctx context.Context) {
	defer close(d.done)

	for {
		interval := d.nextInterval()
		t := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			t.Stop()
			return
		case <-t.C:
		}

		if err := d.rotateOnce(ctx); err != nil {
			slog.Error("country rotation cycle failed", "error", err)
		}
	}
}

func (d *Daemon) rotateOnce(ctx context.Context) error {
	instances := d.rotator.GetInstances()
	if len(instances) == 0 {
		return nil
	}

	d.mu.Lock()
	totalToChange := d.cfg.Country.Rotation.TotalToChange
	accepted := make([]string, len(d.cfg.Country.Accepted))
	copy(accepted, d.cfg.Country.Accepted)
	blacklisted := make([]string, len(d.cfg.Country.Blacklisted))
	copy(blacklisted, d.cfg.Country.Blacklisted)
	d.mu.Unlock()

	if totalToChange <= 0 {
		return nil
	}
	if totalToChange > len(instances) {
		totalToChange = len(instances)
	}

	indices := make([]int, len(instances))
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	for _, idx := range indices[:totalToChange] {
		inst := instances[idx]
		newCountry, err := pickDifferentCountry(inst.Country, accepted, blacklisted)
		if err != nil {
			slog.Error("failed to pick new country",
				"instance", inst.ID,
				"current_country", inst.Country,
				"error", err,
			)
			continue
		}

		if err := d.rotator.RotateInstance(ctx, inst.ID, newCountry); err != nil {
			slog.Error("failed to rotate instance",
				"instance", inst.ID,
				"old_country", inst.Country,
				"new_country", newCountry,
				"error", err,
			)
			continue
		}

		slog.Info("rotated instance country",
			"instance", inst.ID,
			"old_country", inst.Country,
			"new_country", newCountry,
		)
	}

	return nil
}

func (d *Daemon) nextInterval() time.Duration {
	d.mu.Lock()
	base := d.baseInterval
	d.mu.Unlock()

	jitter := time.Duration(rand.Int64N(int64(maxJitter)))
	return base + jitter
}

func (d *Daemon) UpdateConfig(cfg *config.Config) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.cfg = cfg
	d.baseInterval = time.Duration(cfg.Country.Rotation.Interval) * time.Second
}

func pickDifferentCountry(current string, accepted, blacklisted []string) (string, error) {
	filtered := filterBlacklisted(accepted, blacklisted)

	var candidates []string
	for _, c := range filtered {
		if c != current {
			candidates = append(candidates, c)
		}
	}

	if len(candidates) == 0 {
		return "", fmt.Errorf("pickDifferentCountry: no alternative countries available for %q", current)
	}

	return candidates[rand.IntN(len(candidates))], nil
}
