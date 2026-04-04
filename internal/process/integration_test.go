//go:build integration

package process

import (
	"context"
	"testing"
	"time"
)

func TestIntegration_SpawnAndWait(t *testing.T) {
	mgr := NewManager(t.TempDir())
	ctx := context.Background()

	p, err := mgr.Spawn(ctx, "sleep", "/bin/sleep", "1")
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := p.Wait(); err != nil {
		t.Fatalf("Wait: %v", err)
	}

	if p.State() != StateStopped {
		t.Errorf("State = %v, want Stopped", p.State())
	}
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	mgr := NewManager(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := mgr.Spawn(ctx, "sleep", "/bin/sleep", "60")
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	done := make(chan error, 1)
	go func() { done <- p.Wait() }()

	time.Sleep(100 * time.Millisecond)

	if p.State() != StateRunning {
		t.Fatalf("State = %v, want Running before Stop", p.State())
	}

	if err := mgr.Stop(ctx, p); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	if p.State() != StateStopped {
		t.Errorf("State = %v, want Stopped after Stop", p.State())
	}
}

func TestIntegration_StopAll(t *testing.T) {
	mgr := NewManager(t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	p1, err := mgr.Spawn(ctx, "sleep-1", "/bin/sleep", "60")
	if err != nil {
		t.Fatalf("Spawn sleep-1: %v", err)
	}
	p2, err := mgr.Spawn(ctx, "sleep-2", "/bin/sleep", "60")
	if err != nil {
		t.Fatalf("Spawn sleep-2: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	if err := mgr.StopAll(ctx); err != nil {
		t.Fatalf("StopAll: %v", err)
	}

	for i, p := range []*Process{p1, p2} {
		if p.State() != StateStopped {
			t.Errorf("process[%d] State = %v, want Stopped", i, p.State())
		}
	}
}
