package circuit

import (
	"sync"
	"time"
)

type TrafficPattern struct {
	mu             sync.Mutex
	requestTimes   []time.Time
	windowSize     time.Duration
	burstThreshold int
}

func NewTrafficPattern(windowSize time.Duration, burstThreshold int) *TrafficPattern {
	return &TrafficPattern{
		windowSize:     windowSize,
		burstThreshold: burstThreshold,
	}
}

func (tp *TrafficPattern) RecordRequest() {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	now := time.Now()
	tp.requestTimes = append(tp.requestTimes, now)
	tp.pruneLocked(now)
}

func (tp *TrafficPattern) IsBurst() bool {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.pruneLocked(time.Now())
	return len(tp.requestTimes) >= tp.burstThreshold
}

func (tp *TrafficPattern) RequestRate() float64 {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.pruneLocked(time.Now())
	if len(tp.requestTimes) == 0 {
		return 0
	}

	oldest := tp.requestTimes[0]
	windowSeconds := time.Since(oldest).Seconds()
	if windowSeconds <= 0 {
		return float64(len(tp.requestTimes))
	}

	return float64(len(tp.requestTimes)) / windowSeconds
}

func (tp *TrafficPattern) pruneLocked(now time.Time) {
	cutoff := now.Add(-tp.windowSize)
	i := 0
	for i < len(tp.requestTimes) && tp.requestTimes[i].Before(cutoff) {
		i++
	}
	tp.requestTimes = tp.requestTimes[i:]
}
