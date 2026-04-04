package circuit

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestRenewer_AddInstance(t *testing.T) {
	r := NewRenewer()
	r.AddInstance(0, 5999, "/tmp/splitter/tor_data_0/control_auth_cookie")
	r.AddInstance(1, 6000, "/tmp/splitter/tor_data_1/control_auth_cookie")

	if len(r.instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(r.instances))
	}

	if r.instances[0].id != 0 {
		t.Errorf("instance[0].id = %d, want 0", r.instances[0].id)
	}
	if r.instances[0].controlAddr != "127.0.0.1:5999" {
		t.Errorf("instance[0].controlAddr = %q, want %q", r.instances[0].controlAddr, "127.0.0.1:5999")
	}
	if r.instances[1].id != 1 {
		t.Errorf("instance[1].id = %d, want 1", r.instances[1].id)
	}
	if r.instances[1].controlAddr != "127.0.0.1:6000" {
		t.Errorf("instance[1].controlAddr = %q, want %q", r.instances[1].controlAddr, "127.0.0.1:6000")
	}
}

func TestRenewer_RandomInterval(t *testing.T) {
	min := 10 * time.Second
	max := 15 * time.Second

	for i := 0; i < 1000; i++ {
		interval := randomInterval(min, max)
		if interval < min || interval > max {
			t.Errorf("randomInterval() = %v, want range [%v, %v]", interval, min, max)
		}
	}

	uniqueCount := make(map[time.Duration]bool)
	for i := 0; i < 100; i++ {
		interval := randomInterval(min, max)
		uniqueCount[interval] = true
	}
	if len(uniqueCount) < 10 {
		t.Errorf("expected at least 10 unique intervals across 100 samples, got %d", len(uniqueCount))
	}
}

func TestRenewer_RandomInterval_SameMinMax(t *testing.T) {
	d := 10 * time.Second
	interval := randomInterval(d, d)
	if interval != d {
		t.Errorf("randomInterval(10s, 10s) = %v, want %v", interval, d)
	}
}

func TestRenewer_StartStop(t *testing.T) {
	r := NewRenewer()
	r.AddInstance(0, 5999, "/tmp/nonexistent/cookie")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := r.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	if !r.started {
		t.Error("expected started = true")
	}

	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
}

func TestRenewer_StartNoInstances(t *testing.T) {
	r := NewRenewer()
	ctx := context.Background()

	err := r.Start(ctx)
	if err == nil {
		t.Error("expected error when starting with no instances")
	}
}

func TestRenewer_StopBeforeStart(t *testing.T) {
	r := NewRenewer()
	if err := r.Stop(); err != nil {
		t.Fatalf("Stop() before Start() error = %v", err)
	}
}

func TestRenewer_InstanceIntervals(t *testing.T) {
	r := NewRenewer()
	r.AddInstance(0, 5999, "/tmp/cookie")

	inst := r.instances[0]
	if inst.minInterval != defaultMinInterval {
		t.Errorf("minInterval = %v, want %v", inst.minInterval, defaultMinInterval)
	}
	if inst.maxInterval != defaultMaxInterval {
		t.Errorf("maxInterval = %v, want %v", inst.maxInterval, defaultMaxInterval)
	}
}

func TestRenewer_RandomInterval_InvertedMinMax(t *testing.T) {
	min := 15 * time.Second
	max := 10 * time.Second

	interval := randomInterval(min, max)
	if interval != min {
		t.Errorf("randomInterval(15s, 10s) = %v, want %v (min when delta <= 0)", interval, min)
	}
}

func TestRenewer_AddInstanceMultiple(t *testing.T) {
	r := NewRenewer()
	for i := 0; i < 5; i++ {
		r.AddInstance(i, 5999+i, fmt.Sprintf("/tmp/cookie_%d", i))
	}

	if len(r.instances) != 5 {
		t.Fatalf("expected 5 instances, got %d", len(r.instances))
	}

	for i, inst := range r.instances {
		if inst.id != i {
			t.Errorf("instance[%d].id = %d, want %d", i, inst.id, i)
		}
		expectedAddr := fmt.Sprintf("127.0.0.1:%d", 5999+i)
		if inst.controlAddr != expectedAddr {
			t.Errorf("instance[%d].controlAddr = %q, want %q", i, inst.controlAddr, expectedAddr)
		}
	}
}

func TestRenewer_NewRenewerFields(t *testing.T) {
	r := NewRenewer()

	if r.started {
		t.Error("new renewer should not be started")
	}
	if r.cancelFunc != nil {
		t.Error("new renewer should have nil cancelFunc")
	}
	if len(r.instances) != 0 {
		t.Errorf("new renewer should have 0 instances, got %d", len(r.instances))
	}
}
