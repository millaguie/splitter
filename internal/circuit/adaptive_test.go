package circuit

import (
	"testing"
	"time"
)

func TestAdaptiveInterval_Burst(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 5)
	for i := 0; i < 5; i++ {
		tp.RecordRequest()
	}

	for i := 0; i < 100; i++ {
		interval, mode := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
		if mode != "aggressive" {
			t.Errorf("mode = %q, want %q", mode, "aggressive")
		}
		if interval < 3*time.Second || interval > 7*time.Second {
			t.Errorf("burst interval = %v, want range [3s, 7s]", interval)
		}
	}
}

func TestAdaptiveInterval_Moderate(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 20)
	tp.RecordRequest()

	for i := 0; i < 100; i++ {
		interval, mode := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
		if mode != "moderate" {
			t.Errorf("mode = %q, want %q", mode, "moderate")
		}
		if interval < 10*time.Second || interval > 20*time.Second {
			t.Errorf("moderate interval = %v, want range [10s, 20s]", interval)
		}
	}
}

func TestAdaptiveInterval_Idle(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 20)

	for i := 0; i < 100; i++ {
		interval, mode := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
		if mode != "idle" {
			t.Errorf("mode = %q, want %q", mode, "idle")
		}
		if interval < 20*time.Second || interval > 60*time.Second {
			t.Errorf("idle interval = %v, want range [20s, 60s]", interval)
		}
	}
}

func TestAdaptiveInterval_AlwaysPositive(t *testing.T) {
	cases := []struct {
		name string
		tp   *TrafficPattern
	}{
		{"burst", func() *TrafficPattern {
			tp := NewTrafficPattern(30*time.Second, 3)
			tp.RecordRequest()
			tp.RecordRequest()
			tp.RecordRequest()
			return tp
		}()},
		{"moderate", func() *TrafficPattern { tp := NewTrafficPattern(30*time.Second, 20); tp.RecordRequest(); return tp }()},
		{"idle", NewTrafficPattern(30*time.Second, 20)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < 50; i++ {
				interval, _ := AdaptiveInterval(tc.tp, 10*time.Second, 15*time.Second)
				if interval <= 0 {
					t.Errorf("interval = %v, want > 0", interval)
				}
			}
		})
	}
}

func TestAdaptiveInterval_Variance(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 20)

	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		interval, _ := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
		seen[interval] = true
	}

	if len(seen) < 5 {
		t.Errorf("expected at least 5 unique intervals in idle mode over 50 samples, got %d", len(seen))
	}
}

func TestAdaptiveInterval_BurstVariance(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 3)
	tp.RecordRequest()
	tp.RecordRequest()
	tp.RecordRequest()

	seen := make(map[time.Duration]bool)
	for i := 0; i < 50; i++ {
		interval, _ := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
		seen[interval] = true
	}

	if len(seen) < 5 {
		t.Errorf("expected at least 5 unique intervals in burst mode over 50 samples, got %d", len(seen))
	}
}

func TestAdaptiveInterval_Boundary_ModerateRate(t *testing.T) {
	tp := NewTrafficPattern(1*time.Second, 100)
	tp.RecordRequest()

	rate := tp.RequestRate()
	if rate <= 0.1 {
		t.Skipf("rate %f too low for moderate test", rate)
	}

	interval, mode := AdaptiveInterval(tp, 10*time.Second, 15*time.Second)
	if mode != "moderate" {
		t.Errorf("mode = %q for rate %f, want moderate", mode, rate)
	}
	if interval < 10*time.Second || interval > 20*time.Second {
		t.Errorf("interval = %v, want [10s, 20s]", interval)
	}
}
