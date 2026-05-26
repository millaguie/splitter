package health

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"
)

func TestDNSLeakResult_JSON(t *testing.T) {
	tests := []struct {
		name   string
		result DNSLeakResult
	}{
		{
			name: "no leak",
			result: DNSLeakResult{
				LeakDetected: false,
				TorIP:        "1.2.3.4",
				DirectIP:     "",
				TestDomain:   "dnsleaktest.com",
			},
		},
		{
			name: "leak detected",
			result: DNSLeakResult{
				LeakDetected: true,
				TorIP:        "10.0.0.1",
				DirectIP:     "192.168.1.1",
				TestDomain:   "example.com",
				Error:        "",
			},
		},
		{
			name: "with error",
			result: DNSLeakResult{
				LeakDetected: false,
				TorIP:        "",
				DirectIP:     "",
				TestDomain:   "dnsleaktest.com",
				Error:        "connection refused",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.result)
			if err != nil {
				t.Fatalf("json.Marshal error: %v", err)
			}

			var decoded DNSLeakResult
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("json.Unmarshal error: %v", err)
			}

			if decoded.LeakDetected != tt.result.LeakDetected {
				t.Errorf("LeakDetected = %v, want %v", decoded.LeakDetected, tt.result.LeakDetected)
			}
			if decoded.TorIP != tt.result.TorIP {
				t.Errorf("TorIP = %q, want %q", decoded.TorIP, tt.result.TorIP)
			}
			if decoded.DirectIP != tt.result.DirectIP {
				t.Errorf("DirectIP = %q, want %q", decoded.DirectIP, tt.result.DirectIP)
			}
			if decoded.TestDomain != tt.result.TestDomain {
				t.Errorf("TestDomain = %q, want %q", decoded.TestDomain, tt.result.TestDomain)
			}
			if decoded.Error != tt.result.Error {
				t.Errorf("Error = %q, want %q", decoded.Error, tt.result.Error)
			}
		})
	}
}

func TestDNSLeakResult_JSONFieldNames(t *testing.T) {
	r := DNSLeakResult{
		LeakDetected: true,
		TorIP:        "1.2.3.4",
		DirectIP:     "5.6.7.8",
		TestDomain:   "test.com",
		Error:        "some error",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map error: %v", err)
	}

	expectedKeys := []string{"leak_detected", "tor_ip", "direct_ip", "test_domain", "error"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing JSON key %q in output: %s", key, string(data))
		}
	}
}

func TestDNSLeakResult_JSONOmitEmptyError(t *testing.T) {
	r := DNSLeakResult{
		LeakDetected: false,
		TorIP:        "1.2.3.4",
		DirectIP:     "",
		TestDomain:   "test.com",
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal to map error: %v", err)
	}

	if _, ok := raw["error"]; ok {
		t.Errorf("error field should be omitted when empty, got: %s", string(data))
	}
}

type mockSocks5Server struct {
	listener   net.Listener
	responseIP string
	done       chan struct{}
}

func newMockSocks5Server(t *testing.T, responseIP string) *mockSocks5Server {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create mock socks server: %v", err)
	}
	s := &mockSocks5Server{
		listener:   ln,
		responseIP: responseIP,
		done:       make(chan struct{}),
	}
	go s.serve(t)
	return s
}

func (s *mockSocks5Server) Addr() string {
	return s.listener.Addr().String()
}

func (s *mockSocks5Server) Close() {
	_ = s.listener.Close()
	<-s.done
}

func (s *mockSocks5Server) serve(t *testing.T) {
	defer close(s.done)
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handleConn(t, conn)
	}
}

func (s *mockSocks5Server) handleConn(t *testing.T, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	buf := make([]byte, 3)
	if _, err := readFull(conn, buf); err != nil {
		return
	}

	if buf[0] != socks5Version || buf[1] != 1 || buf[2] != socks5NoAuth {
		return
	}

	if _, err := conn.Write([]byte{socks5Version, socks5NoAuth}); err != nil {
		return
	}

	header := make([]byte, 4)
	if _, err := readFull(conn, header); err != nil {
		return
	}

	if header[0] != socks5Version || header[1] != socks5CmdConnect {
		return
	}

	switch header[3] {
	case socks5AtypDomain:
		lenBuf := make([]byte, 1)
		if _, err := readFull(conn, lenBuf); err != nil {
			return
		}
		domainBuf := make([]byte, lenBuf[0])
		if _, err := readFull(conn, domainBuf); err != nil {
			return
		}
	case socks5AtypIPv4:
		ipBuf := make([]byte, 4)
		if _, err := readFull(conn, ipBuf); err != nil {
			return
		}
	case socks5AtypIPv6:
		ipBuf := make([]byte, 16)
		if _, err := readFull(conn, ipBuf); err != nil {
			return
		}
	}

	portBuf := make([]byte, 2)
	if _, err := readFull(conn, portBuf); err != nil {
		return
	}

	ip := net.ParseIP(s.responseIP)
	if ip == nil {
		ip = net.ParseIP("127.0.0.1")
	}

	resp := []byte{socks5Version, 0x00, 0x00}
	if ip4 := ip.To4(); ip4 != nil {
		resp = append(resp, socks5AtypIPv4)
		resp = append(resp, ip4...)
	} else {
		resp = append(resp, socks5AtypIPv6)
		resp = append(resp, ip.To16()...)
	}
	resp = append(resp, portBuf...)

	if _, err := conn.Write(resp); err != nil {
		return
	}

	keepAlive := make([]byte, 1)
	_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _ = conn.Read(keepAlive)
}

func TestDNSLeakTester_Socks5Resolve(t *testing.T) {
	srv := newMockSocks5Server(t, "93.184.216.34")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("example.com")

	ip, err := tester.resolveThroughSocks5(context.Background())
	if err != nil {
		t.Fatalf("resolveThroughSocks5() error = %v", err)
	}

	if ip != "93.184.216.34" {
		t.Errorf("resolveThroughSocks5() = %q, want %q", ip, "93.184.216.34")
	}
}

func TestDNSLeakTester_Socks5ResolveIPv6(t *testing.T) {
	srv := newMockSocks5Server(t, "2606:4700:3030::6815:1a01")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("example.com")

	ip, err := tester.resolveThroughSocks5(context.Background())
	if err != nil {
		t.Fatalf("resolveThroughSocks5() error = %v", err)
	}

	if ip != "2606:4700:3030::6815:1a01" {
		t.Errorf("resolveThroughSocks5() = %q, want %q", ip, "2606:4700:3030::6815:1a01")
	}
}

func TestDNSLeakTester_Socks5ConnectionRefused(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	tester := NewDNSLeakTester(addr)

	_, err = tester.resolveThroughSocks5(context.Background())
	if err == nil {
		t.Fatal("resolveThroughSocks5() expected error for refused connection")
	}
}

func TestDNSLeakTester_Socks5ContextCancelled(t *testing.T) {
	srv := newMockSocks5Server(t, "1.2.3.4")
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	tester := NewDNSLeakTester(srv.Addr())
	_, err := tester.resolveThroughSocks5(ctx)
	if err == nil {
		t.Fatal("resolveThroughSocks5() expected error for cancelled context")
	}
}

func TestDNSLeakTester_Test_NoLeakDirectFails(t *testing.T) {
	srv := newMockSocks5Server(t, "93.184.216.34")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	result, err := tester.Test(context.Background())
	if err != nil {
		t.Fatalf("Test() error = %v", err)
	}

	if result.LeakDetected {
		t.Error("LeakDetected = true, want false (direct resolution fails, tor succeeds)")
	}
	if result.TorIP != "93.184.216.34" {
		t.Errorf("TorIP = %q, want %q", result.TorIP, "93.184.216.34")
	}
	if result.TestDomain != "this-domain-should-not-exist-for-splitter-test.invalid" {
		t.Errorf("TestDomain = %q, want correct domain", result.TestDomain)
	}
}

func TestDNSLeakTester_Test_BothFail(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()

	tester := NewDNSLeakTester(addr)
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	_, err = tester.Test(context.Background())
	if err == nil {
		t.Fatal("Test() expected error when both resolutions fail")
	}
}

func TestNewDNSLeakTester(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	if tester.socksAddr != "127.0.0.1:9050" {
		t.Errorf("socksAddr = %q, want %q", tester.socksAddr, "127.0.0.1:9050")
	}
	if tester.testDomain != defaultDNSLeakTestDomain {
		t.Errorf("testDomain = %q, want %q", tester.testDomain, defaultDNSLeakTestDomain)
	}
}

func TestDNSLeakTester_SetTestDomain(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	tester.SetTestDomain("custom.example.com")
	if tester.testDomain != "custom.example.com" {
		t.Errorf("testDomain = %q, want %q", tester.testDomain, "custom.example.com")
	}
}

func TestPeriodicChecker_StartStop(t *testing.T) {
	srv := newMockSocks5Server(t, "1.2.3.4")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	pc := NewPeriodicChecker(tester, 50*time.Millisecond)

	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	if err := pc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	last := pc.LastResult()
	if last == nil {
		t.Fatal("LastResult() returned nil, expected at least one result")
	}
	if last.TorIP != "1.2.3.4" {
		t.Errorf("LastResult().TorIP = %q, want %q", last.TorIP, "1.2.3.4")
	}
}

func TestPeriodicChecker_MultipleIntervals(t *testing.T) {
	srv := newMockSocks5Server(t, "10.0.0.1")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	pc := NewPeriodicChecker(tester, 50*time.Millisecond)

	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(250 * time.Millisecond)

	if err := pc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	results := pc.Results()
	if len(results) < 2 {
		t.Errorf("len(Results) = %d, want at least 2 results after multiple intervals", len(results))
	}
}

func TestPeriodicChecker_LastResultEmpty(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	pc := NewPeriodicChecker(tester, 5*time.Minute)

	if last := pc.LastResult(); last != nil {
		t.Errorf("LastResult() = %v, want nil before first run", last)
	}
}

func TestPeriodicChecker_ResultsCopy(t *testing.T) {
	srv := newMockSocks5Server(t, "5.5.5.5")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	pc := NewPeriodicChecker(tester, 50*time.Millisecond)
	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	time.Sleep(120 * time.Millisecond)
	if err := pc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	results := pc.Results()
	if len(results) == 0 {
		t.Fatal("Results() returned empty")
	}

	results[0].TorIP = "modified"
	last := pc.LastResult()
	if last.TorIP == "modified" {
		t.Error("Results() should return a copy, not a reference to internal state")
	}
}

func TestPeriodicChecker_StartTwice(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	pc := NewPeriodicChecker(tester, 5*time.Minute)

	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("first Start() error = %v", err)
	}
	defer func() { _ = pc.Stop() }()

	err := pc.Start(context.Background())
	if err == nil {
		t.Fatal("second Start() expected error, got nil")
	}
}

func TestPeriodicChecker_StopWithoutStart(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	pc := NewPeriodicChecker(tester, 5*time.Minute)

	err := pc.Stop()
	if err == nil {
		t.Fatal("Stop() expected error when not started")
	}
}

func TestPeriodicChecker_ContextCancel(t *testing.T) {
	srv := newMockSocks5Server(t, "1.2.3.4")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	ctx, cancel := context.WithCancel(context.Background())

	pc := NewPeriodicChecker(tester, 50*time.Millisecond)
	if err := pc.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	cancel()
	time.Sleep(100 * time.Millisecond)

	last := pc.LastResult()
	if last == nil {
		t.Fatal("LastResult() returned nil, expected at least one result before cancel")
	}
}

func TestPeriodicChecker_DefaultInterval(t *testing.T) {
	tester := NewDNSLeakTester("127.0.0.1:9050")
	pc := NewPeriodicChecker(tester, 0)
	if pc.interval != defaultDNSCheckInterval {
		t.Errorf("interval = %v, want %v for zero input", pc.interval, defaultDNSCheckInterval)
	}

	pc2 := NewPeriodicChecker(tester, -1*time.Second)
	if pc2.interval != defaultDNSCheckInterval {
		t.Errorf("interval = %v, want %v for negative input", pc2.interval, defaultDNSCheckInterval)
	}
}

func TestPeriodicChecker_ResultsTruncation(t *testing.T) {
	srv := newMockSocks5Server(t, "7.7.7.7")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	pc := NewPeriodicChecker(tester, 10*time.Millisecond)

	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	time.Sleep(1500 * time.Millisecond)

	if err := pc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	results := pc.Results()
	if len(results) > 100 {
		t.Errorf("len(Results) = %d, want at most 100", len(results))
	}
}

func TestPeriodicChecker_ConcurrentAccess(t *testing.T) {
	srv := newMockSocks5Server(t, "8.8.8.8")
	defer srv.Close()

	tester := NewDNSLeakTester(srv.Addr())
	tester.SetTestDomain("this-domain-should-not-exist-for-splitter-test.invalid")

	pc := NewPeriodicChecker(tester, 20*time.Millisecond)

	if err := pc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				pc.LastResult()
				pc.Results()
				time.Sleep(5 * time.Millisecond)
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	if err := pc.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	wg.Wait()
}

func TestSocks5ConnectDomain_DomainTooLong(t *testing.T) {
	longDomain := ""
	for i := 0; i < 256; i++ {
		longDomain += "a"
	}

	clientConn, serverConn := net.Pipe()
	defer func() { _ = clientConn.Close() }()
	defer func() { _ = serverConn.Close() }()

	_, err := socks5ConnectDomain(clientConn, longDomain, 80)
	if err == nil {
		t.Fatal("socks5ConnectDomain() expected error for domain > 255 bytes")
	}
}

func TestDNSLeakTester_Socks5InvalidResponse(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, socks5NoAuth})

		header := make([]byte, 4)
		_, _ = readFull(conn, header)

		domainLen := make([]byte, 1)
		_, _ = readFull(conn, domainLen)
		domainBuf := make([]byte, domainLen[0])
		_, _ = readFull(conn, domainBuf)
		portBuf := make([]byte, 2)
		_, _ = readFull(conn, portBuf)

		_, _ = conn.Write([]byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	}()

	tester := NewDNSLeakTester(ln.Addr().String())
	tester.SetTestDomain("example.com")

	_, err = tester.resolveThroughSocks5(context.Background())
	if err == nil {
		t.Fatal("resolveThroughSocks5() expected error for invalid socks version in response")
	}
}

func TestDNSLeakTester_Socks5ErrorCode(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, socks5NoAuth})

		header := make([]byte, 4)
		_, _ = readFull(conn, header)
		domainLen := make([]byte, 1)
		_, _ = readFull(conn, domainLen)
		domainBuf := make([]byte, domainLen[0])
		_, _ = readFull(conn, domainBuf)
		portBuf := make([]byte, 2)
		_, _ = readFull(conn, portBuf)

		resp := []byte{socks5Version, 0x05, 0x00, socks5AtypIPv4, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
		_, _ = conn.Write(resp)
	}()

	tester := NewDNSLeakTester(ln.Addr().String())
	tester.SetTestDomain("example.com")

	_, err = tester.resolveThroughSocks5(context.Background())
	if err == nil {
		t.Fatal("resolveThroughSocks5() expected error for socks error code")
	}
}

func TestSocks5Handshake_InvalidVersion(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{0x04, socks5NoAuth})
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	err = socks5Handshake(conn)
	if err == nil {
		t.Fatal("socks5Handshake() expected error for wrong version")
	}
}

func TestSocks5Handshake_UnsupportedAuth(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, 0xFF})
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	err = socks5Handshake(conn)
	if err == nil {
		t.Fatal("socks5Handshake() expected error for unsupported auth method")
	}
}

func TestSocks5ConnectDomain_DomainAddressType(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, socks5NoAuth})

		header := make([]byte, 4)
		_, _ = readFull(conn, header)

		domainLen := make([]byte, 1)
		_, _ = readFull(conn, domainLen)
		domainBuf := make([]byte, domainLen[0])
		_, _ = readFull(conn, domainBuf)
		portBuf := make([]byte, 2)
		_, _ = readFull(conn, portBuf)

		respDomain := []byte("bound.example.com")
		resp := []byte{socks5Version, 0x00, 0x00, socks5AtypDomain, byte(len(respDomain))}
		resp = append(resp, respDomain...)
		resp = append(resp, portBuf...)
		_, _ = conn.Write(resp)

		keepAlive := make([]byte, 1)
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = conn.Read(keepAlive)
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if err := socks5Handshake(conn); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	addr, err := socks5ConnectDomain(conn, "test.example.com", 80)
	if err != nil {
		t.Fatalf("socks5ConnectDomain() error = %v", err)
	}

	if addr != "bound.example.com" {
		t.Errorf("addr = %q, want %q", addr, "bound.example.com")
	}
}

func TestSocks5ConnectDomain_IPv6AddressType(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, socks5NoAuth})

		header := make([]byte, 4)
		_, _ = readFull(conn, header)

		domainLen := make([]byte, 1)
		_, _ = readFull(conn, domainLen)
		domainBuf := make([]byte, domainLen[0])
		_, _ = readFull(conn, domainBuf)
		portBuf := make([]byte, 2)
		_, _ = readFull(conn, portBuf)

		ip := net.ParseIP("::1").To16()
		resp := []byte{socks5Version, 0x00, 0x00, socks5AtypIPv6}
		resp = append(resp, ip...)
		resp = append(resp, portBuf...)
		_, _ = conn.Write(resp)

		keepAlive := make([]byte, 1)
		_ = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		_, _ = conn.Read(keepAlive)
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if err := socks5Handshake(conn); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	addr, err := socks5ConnectDomain(conn, "test.example.com", 80)
	if err != nil {
		t.Fatalf("socks5ConnectDomain() error = %v", err)
	}

	if addr != "::1" {
		t.Errorf("addr = %q, want %q", addr, "::1")
	}
}

func TestSocks5ConnectDomain_UnsupportedAddrType(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		buf := make([]byte, 3)
		_, _ = readFull(conn, buf)
		_, _ = conn.Write([]byte{socks5Version, socks5NoAuth})

		header := make([]byte, 4)
		_, _ = readFull(conn, header)
		domainLen := make([]byte, 1)
		_, _ = readFull(conn, domainLen)
		domainBuf := make([]byte, domainLen[0])
		_, _ = readFull(conn, domainBuf)
		portBuf := make([]byte, 2)
		_, _ = readFull(conn, portBuf)

		_, _ = conn.Write([]byte{socks5Version, 0x00, 0x00, 0x02, 0x00, 0x00})
	}()

	conn, err := net.DialTimeout("tcp", ln.Addr().String(), 2*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	if err := socks5Handshake(conn); err != nil {
		t.Fatalf("handshake: %v", err)
	}

	_, err = socks5ConnectDomain(conn, "test.example.com", 80)
	if err == nil {
		t.Fatal("socks5ConnectDomain() expected error for unsupported address type")
	}
}

func TestSocks5ConnectDomain_RequestFormat(t *testing.T) {
	server, client := net.Pipe()
	defer func() { _ = server.Close() }()
	defer func() { _ = client.Close() }()

	var received []byte
	done := make(chan struct{})

	go func() {
		defer close(done)
		buf := make([]byte, 256)
		n, _ := server.Read(buf)
		received = buf[:n]

		ip := net.ParseIP("1.2.3.4").To4()
		resp := []byte{socks5Version, 0x00, 0x00, socks5AtypIPv4}
		resp = append(resp, ip...)
		resp = append(resp, 0x00, 0x50)
		_, _ = server.Write(resp)
	}()

	addr, _ := socks5ConnectDomain(client, "test.example.com", 80)
	<-done

	if addr != "1.2.3.4" {
		t.Errorf("addr = %q, want %q", addr, "1.2.3.4")
	}

	if len(received) < 7 {
		t.Fatalf("request too short: %d bytes", len(received))
	}

	if received[0] != socks5Version {
		t.Errorf("version byte = %d, want %d", received[0], socks5Version)
	}
	if received[1] != socks5CmdConnect {
		t.Errorf("cmd byte = %d, want %d", received[1], socks5CmdConnect)
	}
	if received[3] != socks5AtypDomain {
		t.Errorf("atype = %d, want %d", received[3], socks5AtypDomain)
	}
	if received[4] != 16 {
		t.Errorf("domain length = %d, want 16", received[4])
	}

	port := binary.BigEndian.Uint16(received[len(received)-2:])
	if port != 80 {
		t.Errorf("port = %d, want 80", port)
	}
}
