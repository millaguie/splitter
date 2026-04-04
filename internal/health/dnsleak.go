package health

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"
)

const (
	defaultDNSLeakTestDomain = "dnsleaktest.com"
	defaultDNSCheckInterval  = 5 * time.Minute
	socks5Version            = 0x05
	socks5NoAuth             = 0x00
	socks5CmdConnect         = 0x01
	socks5AtypDomain         = 0x03
	socks5AtypIPv4           = 0x01
	socks5AtypIPv6           = 0x04
)

type DNSLeakResult struct {
	LeakDetected bool   `json:"leak_detected"`
	TorIP        string `json:"tor_ip"`
	DirectIP     string `json:"direct_ip"`
	TestDomain   string `json:"test_domain"`
	Error        string `json:"error,omitempty"`
}

type DNSLeakTester struct {
	socksAddr  string
	testDomain string
	dialer     *net.Dialer
}

func NewDNSLeakTester(socksAddr string) *DNSLeakTester {
	return &DNSLeakTester{
		socksAddr:  socksAddr,
		testDomain: defaultDNSLeakTestDomain,
		dialer:     &net.Dialer{Timeout: 15 * time.Second},
	}
}

func (t *DNSLeakTester) SetTestDomain(domain string) {
	t.testDomain = domain
}

func (t *DNSLeakTester) Test(ctx context.Context) (*DNSLeakResult, error) {
	result := &DNSLeakResult{
		TestDomain: t.testDomain,
	}

	var directAddrs []string
	directAddrs, directErr := net.DefaultResolver.LookupHost(ctx, t.testDomain)
	if directErr == nil && len(directAddrs) > 0 {
		result.DirectIP = directAddrs[0]
	}

	torIP, socksErr := t.resolveThroughSocks5(ctx)
	if socksErr != nil {
		if directErr != nil {
			result.Error = fmt.Sprintf("tor resolution: %v; direct resolution: %v", socksErr, directErr)
			return result, fmt.Errorf("Test: both resolutions failed: %w", socksErr)
		}
		result.Error = fmt.Sprintf("tor resolution failed: %v", socksErr)
		return result, fmt.Errorf("Test: tor resolution failed: %w", socksErr)
	}
	result.TorIP = torIP

	if directErr != nil || len(directAddrs) == 0 {
		result.LeakDetected = false
		slog.Info("dns leak test passed", "reason", "direct resolution failed or unavailable", "tor_ip", torIP, "test_domain", t.testDomain)
		return result, nil
	}

	if result.DirectIP == result.TorIP {
		result.LeakDetected = false
		slog.Info("dns leak test passed", "tor_ip", torIP, "direct_ip", result.DirectIP, "test_domain", t.testDomain)
		return result, nil
	}

	result.LeakDetected = true
	slog.Warn("dns leak detected",
		"tor_ip", torIP,
		"direct_ip", result.DirectIP,
		"test_domain", t.testDomain,
	)

	return result, nil
}

func (t *DNSLeakTester) resolveThroughSocks5(ctx context.Context) (string, error) {
	conn, err := t.dialer.DialContext(ctx, "tcp", t.socksAddr)
	if err != nil {
		return "", fmt.Errorf("resolveThroughSocks5: connect to proxy: %w", err)
	}
	defer func() { _ = conn.Close() }()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if err := socks5Handshake(conn); err != nil {
		return "", fmt.Errorf("resolveThroughSocks5: handshake: %w", err)
	}

	boundAddr, err := socks5ConnectDomain(conn, t.testDomain, 80)
	if err != nil {
		return "", fmt.Errorf("resolveThroughSocks5: connect: %w", err)
	}

	return boundAddr, nil
}

func socks5Handshake(conn net.Conn) error {
	if _, err := conn.Write([]byte{socks5Version, 0x01, socks5NoAuth}); err != nil {
		return fmt.Errorf("socks5Handshake: write greeting: %w", err)
	}

	buf := make([]byte, 2)
	if _, err := readFull(conn, buf); err != nil {
		return fmt.Errorf("socks5Handshake: read response: %w", err)
	}

	if buf[0] != socks5Version {
		return fmt.Errorf("socks5Handshake: unexpected version %d", buf[0])
	}
	if buf[1] != socks5NoAuth {
		return fmt.Errorf("socks5Handshake: unsupported auth method %d", buf[1])
	}

	return nil
}

func socks5ConnectDomain(conn net.Conn, domain string, port uint16) (string, error) {
	domainBytes := []byte(domain)
	if len(domainBytes) > 255 {
		return "", fmt.Errorf("socks5ConnectDomain: domain too long (%d bytes)", len(domainBytes))
	}

	req := make([]byte, 0, 4+1+len(domainBytes)+2)
	req = append(req, socks5Version, socks5CmdConnect, 0x00, socks5AtypDomain)
	req = append(req, byte(len(domainBytes)))
	req = append(req, domainBytes...)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, port)
	req = append(req, portBytes...)

	if _, err := conn.Write(req); err != nil {
		return "", fmt.Errorf("socks5ConnectDomain: write request: %w", err)
	}

	header := make([]byte, 4)
	if _, err := readFull(conn, header); err != nil {
		return "", fmt.Errorf("socks5ConnectDomain: read response header: %w", err)
	}

	if header[0] != socks5Version {
		return "", fmt.Errorf("socks5ConnectDomain: unexpected version %d", header[0])
	}
	if header[1] != 0x00 {
		return "", fmt.Errorf("socks5ConnectDomain: socks error code %d", header[1])
	}

	var boundAddr string
	switch header[3] {
	case socks5AtypIPv4:
		ipBuf := make([]byte, 4)
		if _, err := readFull(conn, ipBuf); err != nil {
			return "", fmt.Errorf("socks5ConnectDomain: read ipv4: %w", err)
		}
		boundAddr = net.IP(ipBuf).String()
	case socks5AtypIPv6:
		ipBuf := make([]byte, 16)
		if _, err := readFull(conn, ipBuf); err != nil {
			return "", fmt.Errorf("socks5ConnectDomain: read ipv6: %w", err)
		}
		boundAddr = net.IP(ipBuf).String()
	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := readFull(conn, lenBuf); err != nil {
			return "", fmt.Errorf("socks5ConnectDomain: read domain length: %w", err)
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := readFull(conn, domainBuf); err != nil {
			return "", fmt.Errorf("socks5ConnectDomain: read domain: %w", err)
		}
		boundAddr = string(domainBuf)
	default:
		return "", fmt.Errorf("socks5ConnectDomain: unsupported address type %d", header[3])
	}

	portBuf := make([]byte, 2)
	if _, err := readFull(conn, portBuf); err != nil {
		return "", fmt.Errorf("socks5ConnectDomain: read port: %w", err)
	}

	return boundAddr, nil
}

func readFull(conn net.Conn, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := conn.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

type PeriodicChecker struct {
	tester     *DNSLeakTester
	interval   time.Duration
	results    []DNSLeakResult
	mu         sync.RWMutex
	cancelFunc context.CancelFunc
	done       chan struct{}
}

func NewPeriodicChecker(tester *DNSLeakTester, interval time.Duration) *PeriodicChecker {
	if interval <= 0 {
		interval = defaultDNSCheckInterval
	}
	return &PeriodicChecker{
		tester:   tester,
		interval: interval,
		done:     make(chan struct{}),
	}
}

func (pc *PeriodicChecker) Start(ctx context.Context) error {
	pc.mu.Lock()
	if pc.cancelFunc != nil {
		pc.mu.Unlock()
		return fmt.Errorf("Start: checker already running")
	}
	childCtx, cancel := context.WithCancel(ctx)
	pc.cancelFunc = cancel
	pc.done = make(chan struct{})
	pc.mu.Unlock()

	go pc.run(childCtx)
	return nil
}

func (pc *PeriodicChecker) Stop() error {
	pc.mu.Lock()
	cancel := pc.cancelFunc
	pc.mu.Unlock()

	if cancel == nil {
		return fmt.Errorf("Stop: checker not running")
	}
	cancel()

	<-pc.done

	pc.mu.Lock()
	pc.cancelFunc = nil
	pc.mu.Unlock()

	return nil
}

func (pc *PeriodicChecker) LastResult() *DNSLeakResult {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	if len(pc.results) == 0 {
		return nil
	}
	r := pc.results[len(pc.results)-1]
	return &r
}

func (pc *PeriodicChecker) Results() []DNSLeakResult {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	out := make([]DNSLeakResult, len(pc.results))
	copy(out, pc.results)
	return out
}

func (pc *PeriodicChecker) run(ctx context.Context) {
	defer close(pc.done)

	result, err := pc.tester.Test(ctx)
	if err != nil {
		slog.Warn("periodic dns leak test failed", "error", err)
	} else {
		pc.appendResult(*result)
	}

	ticker := time.NewTicker(pc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := pc.tester.Test(ctx)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				slog.Warn("periodic dns leak test failed", "error", err)
				continue
			}
			pc.appendResult(*result)
		}
	}
}

func (pc *PeriodicChecker) appendResult(r DNSLeakResult) {
	pc.mu.Lock()
	pc.results = append(pc.results, r)
	if len(pc.results) > 100 {
		pc.results = pc.results[len(pc.results)-100:]
	}
	pc.mu.Unlock()
}
