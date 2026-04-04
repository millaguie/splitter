package tor

import (
	"context"
	"testing"
	"time"

	"github.com/user/splitter/internal/process"
)

func TestState_String_Individual(t *testing.T) {
	if got := StateStarting.String(); got != "starting" {
		t.Errorf("StateStarting.String() = %q, want %q", got, "starting")
	}
	if got := StateBootstrapping.String(); got != "bootstrapping" {
		t.Errorf("StateBootstrapping.String() = %q, want %q", got, "bootstrapping")
	}
	if got := StateReady.String(); got != "ready" {
		t.Errorf("StateReady.String() = %q, want %q", got, "ready")
	}
	if got := StateFailed.String(); got != "failed" {
		t.Errorf("StateFailed.String() = %q, want %q", got, "failed")
	}
}

func TestState_Unknown_Value(t *testing.T) {
	s := State(99)
	if got := s.String(); got != "unknown" {
		t.Errorf("State(99).String() = %q, want %q", got, "unknown")
	}
}

func TestBackoffDuration_Zero(t *testing.T) {
	got := backoffDuration(0)
	if got != initialBackoff {
		t.Errorf("backoffDuration(0) = %v, want %v", got, initialBackoff)
	}
}

func TestBackoffDuration_One(t *testing.T) {
	got := backoffDuration(1)
	if got != initialBackoff {
		t.Errorf("backoffDuration(1) = %v, want %v", got, initialBackoff)
	}
}

func TestBackoffDuration_Two(t *testing.T) {
	got := backoffDuration(2)
	if got != 2*time.Second {
		t.Errorf("backoffDuration(2) = %v, want 2s", got)
	}
}

func TestBackoffDuration_Large(t *testing.T) {
	got := backoffDuration(10)
	if got != maxBackoff {
		t.Errorf("backoffDuration(10) = %v, want %v", got, maxBackoff)
	}
}

func TestInstance_StopNilProcess(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	ctx := context.Background()
	if err := inst.Stop(ctx); err != nil {
		t.Errorf("Stop() on unstarted instance returned error: %v", err)
	}
}

func TestInstance_WaitNilProcess(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(0, "{US}", cfg, v, procMgr)

	if err := inst.Wait(); err != nil {
		t.Errorf("Wait() on unstarted instance returned error: %v", err)
	}
}

func TestInstance_ProcessName_WithConfig(t *testing.T) {
	cfg := testConfig(t)
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}

	inst := NewInstance(7, "{US}", cfg, v, procMgr)

	got := inst.processName()
	want := "tor-7"
	if got != want {
		t.Errorf("processName() = %q, want %q", got, want)
	}
}

func TestManager_GetInstances_ReturnsCopy(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 2
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}", "{DE}"})

	a := mgr.GetInstances()
	b := mgr.GetInstances()

	if len(a) != len(b) {
		t.Fatalf("GetInstances() lengths differ: %d vs %d", len(a), len(b))
	}

	if &a[0] == &b[0] {
		t.Error("GetInstances() returned slices sharing same backing array; expected independent copies")
	}
}

func TestManager_GetInstances_PreservesPointers(t *testing.T) {
	cfg := testConfig(t)
	cfg.Instances.PerCountry = 1
	cfg.Tor.StartSocksPort = 4999
	cfg.Tor.StartControlPort = 5999
	cfg.Tor.StartHTTPPort = 5199

	procMgr := process.NewManager(cfg.Paths.TempFiles)
	mgr := NewManager(cfg, procMgr)

	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	mgr.CreateFromVersion(v, []string{"{US}"})

	a := mgr.GetInstances()
	b := mgr.GetInstances()

	if len(a) != 1 || len(b) != 1 {
		t.Fatalf("expected 1 instance, got %d and %d", len(a), len(b))
	}

	if a[0] != b[0] {
		t.Error("GetInstances() elements point to different Instance objects; expected same underlying pointers")
	}
}
