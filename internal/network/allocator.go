package network

import (
	"fmt"
	"log/slog"
	"net"
	"sync"
)

const maxPortScan = 1000

type Allocator struct {
	mu     sync.Mutex
	allocd map[int]struct{}
}

func NewAllocator() *Allocator {
	return &Allocator{
		allocd: make(map[int]struct{}),
	}
}

func (a *Allocator) AllocatePort(preferredPort int) (int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.allocatePortLocked(preferredPort)
}

func (a *Allocator) allocatePortLocked(preferredPort int) (int, error) {
	for offset := 0; offset < maxPortScan; offset++ {
		candidate := preferredPort + offset
		if candidate > 65535 {
			break
		}
		if _, taken := a.allocd[candidate]; taken {
			continue
		}
		if !isAvailable(candidate) {
			slog.Debug("port occupied, skipping", "port", candidate)
			continue
		}
		a.allocd[candidate] = struct{}{}
		slog.Debug("allocated port", "port", candidate)
		return candidate, nil
	}
	return 0, fmt.Errorf("AllocatePort: no available port in range %d-%d", preferredPort, preferredPort+maxPortScan)
}

func (a *Allocator) AllocateN(preferredPort int, count int) (int, error) {
	if count <= 0 {
		return 0, fmt.Errorf("AllocateN: count must be positive, got %d", count)
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	for base := preferredPort; base+count-1 <= 65535 && base < preferredPort+maxPortScan; base++ {
		allFree := true
		for i := 0; i < count; i++ {
			p := base + i
			if _, taken := a.allocd[p]; taken {
				allFree = false
				break
			}
			if !isAvailable(p) {
				slog.Debug("port occupied during consecutive scan, skipping", "port", p)
				allFree = false
				break
			}
		}
		if !allFree {
			continue
		}
		for i := 0; i < count; i++ {
			a.allocd[base+i] = struct{}{}
		}
		slog.Debug("allocated consecutive ports", "start", base, "count", count)
		return base, nil
	}
	return 0, fmt.Errorf("AllocateN: no %d consecutive ports available starting from %d", count, preferredPort)
}

func (a *Allocator) Release(port int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.allocd, port)
	slog.Debug("released port", "port", port)
}

func (a *Allocator) ReleaseAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.allocd = make(map[int]struct{})
	slog.Debug("released all ports")
}

func (a *Allocator) Allocated() []int {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make([]int, 0, len(a.allocd))
	for p := range a.allocd {
		out = append(out, p)
	}
	return out
}

func isAvailable(port int) bool {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	_ = l.Close()
	return true
}
