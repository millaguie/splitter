package country

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/user/splitter/internal/config"
)

type rotationRecord struct {
	ID      int
	Country string
}

type mockRotator struct {
	mu        sync.Mutex
	instances []InstanceInfo
	rotated   []rotationRecord
}

func newMockRotator(instances []InstanceInfo) *mockRotator {
	inst := make([]InstanceInfo, len(instances))
	copy(inst, instances)
	return &mockRotator{instances: inst}
}

func (m *mockRotator) GetInstances() []InstanceInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]InstanceInfo, len(m.instances))
	copy(out, m.instances)
	return out
}

func (m *mockRotator) RotateInstance(_ context.Context, id int, newCountry string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rotated = append(m.rotated, rotationRecord{ID: id, Country: newCountry})
	for i := range m.instances {
		if m.instances[i].ID == id {
			m.instances[i].Country = newCountry
			break
		}
	}
	return nil
}

func (m *mockRotator) getRotated() []rotationRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]rotationRecord, len(m.rotated))
	copy(out, m.rotated)
	return out
}

func TestDaemon_New(t *testing.T) {
	cfg := testDaemonConfig()
	rotator := newMockRotator(nil)
	d := NewDaemon(cfg, rotator)

	if d == nil {
		t.Fatal("NewDaemon returned nil")
	}
	if d.baseInterval != 120*time.Second {
		t.Errorf("baseInterval = %v, want %v", d.baseInterval, 120*time.Second)
	}
}

func TestDaemon_RotationChangesCountry(t *testing.T) {
	cfg := testDaemonConfig()
	cfg.Country.Rotation.TotalToChange = 3

	originals := map[int]string{0: "{US}", 1: "{DE}", 2: "{FR}"}
	rotator := newMockRotator([]InstanceInfo{
		{ID: 0, Country: "{US}"},
		{ID: 1, Country: "{DE}"},
		{ID: 2, Country: "{FR}"},
	})

	d := NewDaemon(cfg, rotator)
	ctx := context.Background()

	if err := d.rotateOnce(ctx); err != nil {
		t.Fatalf("rotateOnce() error = %v", err)
	}

	rotated := rotator.getRotated()
	if len(rotated) != 3 {
		t.Fatalf("expected 3 rotations, got %d", len(rotated))
	}

	for _, r := range rotated {
		original := originals[r.ID]
		if r.Country == original {
			t.Errorf("instance %d: country did not change (still %q)", r.ID, r.Country)
		}
	}
}

func TestDaemon_Jitter(t *testing.T) {
	cfg := testDaemonConfig()
	rotator := newMockRotator(nil)
	d := NewDaemon(cfg, rotator)

	uniqueIntervals := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		interval := d.nextInterval()
		uniqueIntervals[interval] = true

		min := d.baseInterval
		max := d.baseInterval + maxJitter
		if interval < min || interval > max {
			t.Errorf("interval %v out of range [%v, %v]", interval, min, max)
		}
	}

	if len(uniqueIntervals) < 10 {
		t.Errorf("expected at least 10 unique intervals across 100 samples, got %d", len(uniqueIntervals))
	}
}

func TestDaemon_DisabledRotation(t *testing.T) {
	cfg := testDaemonConfig()
	cfg.Country.Rotation.Enabled = false

	rotator := newMockRotator([]InstanceInfo{
		{ID: 0, Country: "{US}"},
	})

	d := NewDaemon(cfg, rotator)
	ctx := context.Background()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	rotated := rotator.getRotated()
	if len(rotated) != 0 {
		t.Errorf("expected no rotations when disabled, got %d", len(rotated))
	}
}

func TestDaemon_StopBeforeStart(t *testing.T) {
	cfg := testDaemonConfig()
	rotator := newMockRotator(nil)
	d := NewDaemon(cfg, rotator)

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() before Start() error = %v", err)
	}
}

func TestDaemon_StartStop(t *testing.T) {
	cfg := testDaemonConfig()
	cfg.Country.Rotation.Interval = 1

	rotator := newMockRotator([]InstanceInfo{
		{ID: 0, Country: "{US}"},
		{ID: 1, Country: "{DE}"},
	})

	d := NewDaemon(cfg, rotator)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	if !d.started {
		t.Error("expected started to be true")
	}
}

func TestDaemon_EmptyInstances(t *testing.T) {
	cfg := testDaemonConfig()
	rotator := newMockRotator(nil)

	d := NewDaemon(cfg, rotator)
	ctx := context.Background()

	if err := d.rotateOnce(ctx); err != nil {
		t.Fatalf("rotateOnce() with empty instances error = %v", err)
	}

	rotated := rotator.getRotated()
	if len(rotated) != 0 {
		t.Errorf("expected no rotations with empty instances, got %d", len(rotated))
	}
}

func TestDaemon_UpdateConfig(t *testing.T) {
	cfg := testDaemonConfig()
	cfg.Country.Rotation.Interval = 120
	rotator := newMockRotator(nil)
	d := NewDaemon(cfg, rotator)

	if d.baseInterval != 120*time.Second {
		t.Errorf("initial baseInterval = %v, want %v", d.baseInterval, 120*time.Second)
	}

	newCfg := testDaemonConfig()
	newCfg.Country.Rotation.Interval = 60
	d.UpdateConfig(newCfg)

	if d.baseInterval != 60*time.Second {
		t.Errorf("updated baseInterval = %v, want %v", d.baseInterval, 60*time.Second)
	}
}

func TestDaemon_UpdateConfig_ThreadSafe(t *testing.T) {
	cfg := testDaemonConfig()
	cfg.Country.Rotation.Interval = 1
	rotator := newMockRotator([]InstanceInfo{
		{ID: 0, Country: "{US}"},
	})
	d := NewDaemon(cfg, rotator)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := d.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func(interval int) {
			defer wg.Done()
			c := testDaemonConfig()
			c.Country.Rotation.Interval = interval
			d.UpdateConfig(c)
		}(i + 10)
	}
	wg.Wait()

	if err := d.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func testDaemonConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Country.Accepted = []string{
		"{AU}", "{AT}", "{BE}", "{BG}", "{CA}", "{CZ}", "{DK}",
		"{FI}", "{FR}", "{DE}", "{HU}", "{IS}", "{LV}", "{LT}",
		"{LU}", "{MD}", "{NL}", "{NO}", "{PA}", "{PL}", "{RO}",
		"{RU}", "{SC}", "{SG}", "{SK}", "{ES}", "{SE}", "{CH}",
		"{TR}", "{UA}", "{GB}", "{US}",
	}
	cfg.Country.Blacklisted = nil
	cfg.Country.Rotation.Enabled = true
	cfg.Country.Rotation.Interval = 120
	cfg.Country.Rotation.TotalToChange = 10
	return cfg
}
