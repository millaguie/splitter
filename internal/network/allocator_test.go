package network

import (
	"fmt"
	"net"
	"sort"
	"sync"
	"testing"
)

func TestNewAllocator(t *testing.T) {
	a := NewAllocator()
	if a == nil {
		t.Fatal("NewAllocator returned nil")
	}
	if got := len(a.Allocated()); got != 0 {
		t.Fatalf("expected 0 allocated, got %d", got)
	}
}

func TestAllocatePort(t *testing.T) {
	a := NewAllocator()
	port, err := a.AllocatePort(50000)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	if port < 50000 || port > 50000+maxPortScan {
		t.Fatalf("port %d out of expected range", port)
	}
	allocd := a.Allocated()
	if len(allocd) != 1 {
		t.Fatalf("expected 1 allocated port, got %d", len(allocd))
	}
	if allocd[0] != port {
		t.Fatalf("expected allocated port %d, got %d", port, allocd[0])
	}
}

func TestAllocatePort_doubleAllocate(t *testing.T) {
	a := NewAllocator()
	p1, err := a.AllocatePort(50000)
	if err != nil {
		t.Fatalf("first AllocatePort: %v", err)
	}
	p2, err := a.AllocatePort(50000)
	if err != nil {
		t.Fatalf("second AllocatePort: %v", err)
	}
	if p1 == p2 {
		t.Fatalf("should not allocate same port twice: %d", p1)
	}
	allocd := a.Allocated()
	if len(allocd) != 2 {
		t.Fatalf("expected 2 allocated ports, got %d", len(allocd))
	}
}

func TestAllocatePort_concurrent(t *testing.T) {
	a := NewAllocator()
	const n = 50
	ports := make([]int, n)
	errs := make([]error, n)
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			ports[idx], errs[idx] = a.AllocatePort(45000)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	seen := make(map[int]struct{})
	for _, p := range ports {
		if _, dup := seen[p]; dup {
			t.Fatalf("duplicate port allocated: %d", p)
		}
		seen[p] = struct{}{}
	}
	if len(seen) != n {
		t.Fatalf("expected %d unique ports, got %d", n, len(seen))
	}
}

func TestAllocateN(t *testing.T) {
	a := NewAllocator()
	base, err := a.AllocateN(48000, 3)
	if err != nil {
		t.Fatalf("AllocateN: %v", err)
	}
	allocd := a.Allocated()
	if len(allocd) != 3 {
		t.Fatalf("expected 3 allocated ports, got %d", len(allocd))
	}
	sort.Ints(allocd)
	for i := 0; i < 3; i++ {
		if allocd[i] != base+i {
			t.Fatalf("expected port %d, got %d", base+i, allocd[i])
		}
	}
}

func TestAllocateN_concurrent(t *testing.T) {
	a := NewAllocator()
	const goroutines = 20
	const count = 3
	results := make([]int, goroutines)
	errs := make([]error, goroutines)
	var wg sync.WaitGroup

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			results[idx], errs[idx] = a.AllocateN(42000, count)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("goroutine %d: %v", i, err)
		}
	}

	allPorts := make(map[int]struct{})
	for _, base := range results {
		for j := 0; j < count; j++ {
			p := base + j
			if _, dup := allPorts[p]; dup {
				t.Fatalf("duplicate port in consecutive range: %d", p)
			}
			allPorts[p] = struct{}{}
		}
	}
	if len(allPorts) != goroutines*count {
		t.Fatalf("expected %d unique ports, got %d", goroutines*count, len(allPorts))
	}
}

func TestAllocateN_invalidCount(t *testing.T) {
	a := NewAllocator()
	_, err := a.AllocateN(50000, 0)
	if err == nil {
		t.Fatal("expected error for count=0")
	}
	_, err = a.AllocateN(50000, -1)
	if err == nil {
		t.Fatal("expected error for negative count")
	}
}

func TestRelease(t *testing.T) {
	a := NewAllocator()
	port, err := a.AllocatePort(49000)
	if err != nil {
		t.Fatalf("AllocatePort: %v", err)
	}
	a.Release(port)
	if len(a.Allocated()) != 0 {
		t.Fatal("expected 0 allocated after release")
	}
	p2, err := a.AllocatePort(port)
	if err != nil {
		t.Fatalf("re-allocate after release: %v", err)
	}
	if p2 != port {
		t.Fatalf("expected re-allocation of same port %d, got %d", port, p2)
	}
}

func TestReleaseAll(t *testing.T) {
	a := NewAllocator()
	for i := 0; i < 5; i++ {
		_, err := a.AllocatePort(47000)
		if err != nil {
			t.Fatalf("AllocatePort %d: %v", i, err)
		}
	}
	if len(a.Allocated()) != 5 {
		t.Fatalf("expected 5 allocated, got %d", len(a.Allocated()))
	}
	a.ReleaseAll()
	if len(a.Allocated()) != 0 {
		t.Fatalf("expected 0 after ReleaseAll, got %d", len(a.Allocated()))
	}
}

func TestAllocatePort_skipOccupied(t *testing.T) {
	a := NewAllocator()
	blockPort := 46500

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", blockPort))
	if err != nil {
		t.Skipf("cannot bind port %d: %v", blockPort, err)
	}
	defer func() { _ = l.Close() }()

	port, err := a.AllocatePort(blockPort)
	if err != nil {
		t.Fatalf("AllocatePort with occupied start: %v", err)
	}
	if port == blockPort {
		t.Fatalf("should have skipped occupied port %d", blockPort)
	}
	if port < blockPort || port > blockPort+maxPortScan {
		t.Fatalf("port %d out of expected range", port)
	}
}
