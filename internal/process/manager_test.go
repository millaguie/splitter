package process

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSpawn_Success(t *testing.T) {
	m := NewManager(t.TempDir())
	defer func() { _ = m.StopAll(t.Context()) }()

	p, err := m.Spawn(t.Context(), "test-sleep", "sleep", "10")
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	if p.State() != StateRunning {
		t.Errorf("State() = %v, want StateRunning", p.State())
	}

	if p.Pid() <= 0 {
		t.Errorf("Pid() = %d, want > 0", p.Pid())
	}

	if p.Name != "test-sleep" {
		t.Errorf("Name = %q, want %q", p.Name, "test-sleep")
	}
}

func TestSpawn_InvalidBinary(t *testing.T) {
	m := NewManager("")

	_, err := m.Spawn(t.Context(), "bad", "/nonexistent/binary_xyz")
	if err == nil {
		t.Fatal("Spawn() expected error for invalid binary, got nil")
	}
}

func TestSpawn_CancelledContext(t *testing.T) {
	m := NewManager("")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	_, err := m.Spawn(ctx, "cancelled", "sleep", "10")
	if err == nil {
		t.Fatal("Spawn() expected error with cancelled context, got nil")
	}
}

func TestStop_Graceful(t *testing.T) {
	m := NewManager(t.TempDir())

	p, err := m.Spawn(t.Context(), "graceful", "sleep", "60")
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	start := time.Now()
	if err := m.Stop(t.Context(), p); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	elapsed := time.Since(start)

	if elapsed > 6*time.Second {
		t.Errorf("Stop() took %v, expected < 6s (graceful SIGTERM)", elapsed)
	}

	if p.State() != StateStopped {
		t.Errorf("State() = %v, want StateStopped", p.State())
	}
}

func TestStop_AlreadyStopped(t *testing.T) {
	m := NewManager(t.TempDir())

	p, err := m.Spawn(t.Context(), "quick", "true")
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	if err := p.Wait(); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if err := m.Stop(t.Context(), p); err != nil {
		t.Fatalf("Stop() on already-stopped process returned error: %v", err)
	}
}

func TestStop_Nil(t *testing.T) {
	m := NewManager("")
	if err := m.Stop(t.Context(), nil); err != nil {
		t.Fatalf("Stop(nil) returned error: %v", err)
	}
}

func TestStopAll(t *testing.T) {
	m := NewManager(t.TempDir())

	p1, err := m.Spawn(t.Context(), "s1", "sleep", "60")
	if err != nil {
		t.Fatalf("Spawn s1 error = %v", err)
	}
	p2, err := m.Spawn(t.Context(), "s2", "sleep", "60")
	if err != nil {
		t.Fatalf("Spawn s2 error = %v", err)
	}
	p3, err := m.Spawn(t.Context(), "s3", "sleep", "60")
	if err != nil {
		t.Fatalf("Spawn s3 error = %v", err)
	}

	if err := m.StopAll(t.Context()); err != nil {
		t.Fatalf("StopAll() error = %v", err)
	}

	for i, p := range [](*Process){p1, p2, p3} {
		if p.State() != StateStopped {
			t.Errorf("process[%d] State() = %v, want StateStopped", i, p.State())
		}
	}
}

func TestStop_SIGKILLTimeout(t *testing.T) {
	m := NewManager(t.TempDir())

	p, err := m.Spawn(t.Context(), "sigterm-proof",
		"sh", "-c", "trap '' TERM; while true; do sleep 60; done")
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	start := time.Now()
	if err := m.Stop(t.Context(), p); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	elapsed := time.Since(start)

	if elapsed < gracefulShutdownTimeout {
		t.Errorf("Stop() took %v, expected >= %v (SIGKILL escalation)", elapsed, gracefulShutdownTimeout)
	}

	if elapsed > gracefulShutdownTimeout+3*time.Second {
		t.Errorf("Stop() took %v, expected < %v", elapsed, gracefulShutdownTimeout+3*time.Second)
	}

	if p.State() != StateStopped {
		t.Errorf("State() = %v, want StateStopped", p.State())
	}
}

func TestCleanup(t *testing.T) {
	dir, err := os.MkdirTemp("", "splitter-cleanup-test-*")
	if err != nil {
		t.Fatal(err)
	}

	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	m := NewManager(dir)
	if err := m.Cleanup(); err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}

	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("expected temp dir %q to be removed", dir)
	}
}

func TestCleanup_EmptyDir(t *testing.T) {
	m := NewManager("")
	if err := m.Cleanup(); err != nil {
		t.Fatalf("Cleanup() with empty dir returned error: %v", err)
	}
}

func TestCleanup_NonexistentDir(t *testing.T) {
	m := NewManager("/tmp/splitter-nonexistent-dir-xyz-999")
	if err := m.Cleanup(); err != nil {
		t.Fatalf("Cleanup() with nonexistent dir returned error: %v", err)
	}
}

func TestList(t *testing.T) {
	m := NewManager(t.TempDir())
	defer func() { _ = m.StopAll(t.Context()) }()

	if procs := m.List(); len(procs) != 0 {
		t.Errorf("List() returned %d processes, want 0", len(procs))
	}

	_, err := m.Spawn(t.Context(), "s1", "sleep", "10")
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.Spawn(t.Context(), "s2", "sleep", "10")
	if err != nil {
		t.Fatal(err)
	}

	procs := m.List()
	if len(procs) != 2 {
		t.Errorf("List() returned %d processes, want 2", len(procs))
	}
}

func TestWait(t *testing.T) {
	m := NewManager(t.TempDir())

	p, err := m.Spawn(t.Context(), "quick-exit", "true")
	if err != nil {
		t.Fatalf("Spawn() error = %v", err)
	}

	if err := m.Wait(p); err != nil {
		t.Fatalf("Wait() error = %v", err)
	}

	if p.State() != StateStopped {
		t.Errorf("State() = %v, want StateStopped", p.State())
	}
}

func TestWait_Nil(t *testing.T) {
	m := NewManager("")
	if err := m.Wait(nil); err == nil {
		t.Fatal("Wait(nil) expected error, got nil")
	}
}
