package circuit

import (
	"time"
)

func AdaptiveInterval(pattern *TrafficPattern, baseMin, baseMax time.Duration) (time.Duration, string) {
	if pattern.IsBurst() {
		interval := randomInterval(3*time.Second, 7*time.Second)
		return interval, "aggressive"
	}

	rate := pattern.RequestRate()
	if rate > 0.1 {
		interval := randomInterval(10*time.Second, 20*time.Second)
		return interval, "moderate"
	}

	interval := randomInterval(20*time.Second, 60*time.Second)
	return interval, "idle"
}
