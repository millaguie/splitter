package circuit

import (
	"testing"
	"time"
)

func TestTrafficPattern_IsBurst_NoRequests(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 20)
	if tp.IsBurst() {
		t.Error("IsBurst() = true with no requests, want false")
	}
}

func TestTrafficPattern_IsBurst_ThresholdExceeded(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 5)
	for i := 0; i < 5; i++ {
		tp.RecordRequest()
	}
	if !tp.IsBurst() {
		t.Error("IsBurst() = false after 5 requests with threshold 5, want true")
	}
}

func TestTrafficPattern_IsBurst_BelowThreshold(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 10)
	for i := 0; i < 9; i++ {
		tp.RecordRequest()
	}
	if tp.IsBurst() {
		t.Error("IsBurst() = true after 9 requests with threshold 10, want false")
	}
}

func TestTrafficPattern_RecordRequest_PruneOld(t *testing.T) {
	tp := NewTrafficPattern(100*time.Millisecond, 10)

	tp.RecordRequest()
	time.Sleep(150 * time.Millisecond)

	tp.RecordRequest()

	tp.mu.Lock()
	count := len(tp.requestTimes)
	tp.mu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 request after prune, got %d", count)
	}
}

func TestTrafficPattern_RequestRate_NoRequests(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 20)
	rate := tp.RequestRate()
	if rate != 0 {
		t.Errorf("RequestRate() = %f, want 0 with no requests", rate)
	}
}

func TestTrafficPattern_RequestRate_Accuracy(t *testing.T) {
	tp := NewTrafficPattern(10*time.Second, 100)

	for i := 0; i < 10; i++ {
		tp.RecordRequest()
	}

	rate := tp.RequestRate()
	if rate <= 0 {
		t.Errorf("RequestRate() = %f, want > 0", rate)
	}

	tp.mu.Lock()
	count := len(tp.requestTimes)
	tp.mu.Unlock()

	if count != 10 {
		t.Errorf("expected 10 requests in window, got %d", count)
	}
}

func TestTrafficPattern_RequestRate_AllPruned(t *testing.T) {
	tp := NewTrafficPattern(50*time.Millisecond, 10)

	tp.RecordRequest()
	time.Sleep(100 * time.Millisecond)

	rate := tp.RequestRate()
	if rate != 0 {
		t.Errorf("RequestRate() = %f after all pruned, want 0", rate)
	}
}

func TestTrafficPattern_ConcurrentAccess(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 100)
	done := make(chan struct{})

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				tp.RecordRequest()
				_ = tp.IsBurst()
				_ = tp.RequestRate()
			}
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestTrafficPattern_WindowPruningBoundary(t *testing.T) {
	tp := NewTrafficPattern(200*time.Millisecond, 10)

	tp.RecordRequest()
	time.Sleep(100 * time.Millisecond)
	tp.RecordRequest()
	time.Sleep(100 * time.Millisecond)

	tp.RecordRequest()

	tp.mu.Lock()
	count := len(tp.requestTimes)
	tp.mu.Unlock()

	if count == 0 {
		t.Error("expected at least 1 request remaining after staggered recording")
	}
}

func TestTrafficPattern_IsBurst_ExactThreshold(t *testing.T) {
	tp := NewTrafficPattern(30*time.Second, 3)
	tp.RecordRequest()
	tp.RecordRequest()
	if tp.IsBurst() {
		t.Error("IsBurst() = true with 2 requests and threshold 3")
	}
	tp.RecordRequest()
	if !tp.IsBurst() {
		t.Error("IsBurst() = false with 3 requests and threshold 3")
	}
}
