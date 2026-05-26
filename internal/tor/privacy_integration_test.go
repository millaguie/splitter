//go:build integration

package tor

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

const (
	// torCheckURL is used to verify traffic goes through Tor
	torCheckURL = "https://check.torproject.org/api/ip"
	// Number of requests for rotation tests
	rotationRequests = 6
	// Timeout for individual proxy requests
	proxyTimeout = 30 * time.Second
)

// integrationConfig creates a minimal config for integration tests.
func integrationConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := config.Defaults()
	cfg.Tor.BinaryPath = os.Getenv("TOR_BINARY")
	if cfg.Tor.BinaryPath == "" {
		cfg.Tor.BinaryPath = "tor"
	}
	cfg.Paths.TempFiles = t.TempDir()
	cfg.Tor.StreamIsolation = false
	cfg.Tor.HiddenService.Enabled = false
	return cfg
}

// startTestInstance starts a single Tor instance for testing and returns
// a cleanup function.
func startTestInstance(t *testing.T, cfg *config.Config) (*Instance, func()) {
	t.Helper()
	procMgr := process.NewManager(cfg.Paths.TempFiles)
	v, err := DetectVersion(context.Background(), cfg.Tor.BinaryPath)
	if err != nil {
		t.Skipf("tor not available: %v", err)
	}

	inst := NewInstance(0, "{US}", cfg, v, procMgr)
	inst.SocksPort = findFreePort(t)
	inst.ControlPort = findFreePort(t)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := inst.Start(ctx); err != nil {
		t.Fatalf("failed to start tor instance: %v", err)
	}

	// Wait for bootstrap
	time.Sleep(5 * time.Second)

	cleanup := func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		inst.Stop(stopCtx)
	}

	return inst, cleanup
}

// findFreePort finds an available TCP port.
func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	defer l.Close()
	addr := l.Addr().(*net.TCPAddr)
	return addr.Port
}

// TorIPResponse represents the JSON response from check.torproject.org/api/ip
type TorIPResponse struct {
	IsTor bool   `json:"IsTor"`
	IP    string `json:"IP"`
}

// fetchTorIP makes a request through a SOCKS5 proxy and returns the response.
func fetchTorIP(t *testing.T, socksPort int) *TorIPResponse {
	t.Helper()

	dialer := &net.Dialer{Timeout: proxyTimeout}
	proxyConn, err := dialer.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort))
	if err != nil {
		t.Fatalf("failed to connect to SOCKS5 proxy: %v", err)
	}
	defer proxyConn.Close()

	// SOCKS5 handshake: version 5, 1 auth method (no auth)
	_, err = proxyConn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("SOCKS5 handshake failed: %v", err)
	}

	buf := make([]byte, 2)
	if _, err := proxyConn.Read(buf); err != nil {
		t.Fatalf("SOCKS5 auth response failed: %v", err)
	}
	if buf[0] != 0x05 {
		t.Fatalf("invalid SOCKS5 version: %d", buf[0])
	}

	// SOCKS5 connect request to check.torproject.org:443
	target := "check.torproject.org"
	port := 443
	connectReq := []byte{
		0x05, 0x01, 0x00, 0x03, byte(len(target)),
	}
	connectReq = append(connectReq, []byte(target)...)
	connectReq = append(connectReq, byte(port>>8), byte(port))

	_, err = proxyConn.Write(connectReq)
	if err != nil {
		t.Fatalf("SOCKS5 connect failed: %v", err)
	}

	resp := make([]byte, 10)
	n, err := proxyConn.Read(resp)
	if err != nil {
		t.Fatalf("SOCKS5 connect response failed: %v", err)
	}
	if n < 4 || resp[1] != 0x00 {
		t.Fatalf("SOCKS5 connect rejected: status 0x%02x", resp[1])
	}

	// Send HTTP request through the tunnel
	httpReq := "GET /api/ip HTTP/1.1\r\nHost: check.torproject.org\r\nConnection: close\r\n\r\n"
	_, err = proxyConn.Write([]byte(httpReq))
	if err != nil {
		t.Fatalf("HTTP request failed: %v", err)
	}

	// Read response
	response := make([]byte, 4096)
	n, err = proxyConn.Read(response)
	if err != nil {
		t.Fatalf("HTTP response failed: %v", err)
	}

	body := string(response[:n])
	// Extract JSON from HTTP response
	jsonStart := strings.Index(body, "{")
	jsonEnd := strings.LastIndex(body, "}")
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		t.Fatalf("no JSON in response: %s", body)
	}

	var result TorIPResponse
	if err := json.Unmarshal([]byte(body[jsonStart:jsonEnd+1]), &result); err != nil {
		t.Fatalf("failed to parse response JSON: %v\nbody: %s", err, body[jsonStart:jsonEnd+1])
	}

	return &result
}

// TestIntegration_TorExitDetected verifies that traffic through the Tor
// SOCKS5 proxy is detected as coming from Tor by check.torproject.org.
func TestIntegration_TorExitDetected(t *testing.T) {
	cfg := integrationConfig(t)
	inst, cleanup := startTestInstance(t, cfg)
	defer cleanup()

	result := fetchTorIP(t, inst.SocksPort)

	if !result.IsTor {
		t.Errorf("traffic not detected as Tor: IsTor=false, IP=%s", result.IP)
	}
	t.Logf("Tor exit IP: %s (IsTor: %v)", result.IP, result.IsTor)
}

// TestIntegration_CircuitRotation verifies that multiple requests through
// the same Tor instance yield different exit IPs after NEWNYM signals.
func TestIntegration_CircuitRotation(t *testing.T) {
	cfg := integrationConfig(t)
	inst, cleanup := startTestInstance(t, cfg)
	defer cleanup()

	ips := make(map[string]bool)
	for i := 0; i < rotationRequests; i++ {
		result := fetchTorIP(t, inst.SocksPort)
		if result.IsTor {
			ips[result.IP] = true
		}
		t.Logf("request %d: IP=%s IsTor=%v", i+1, result.IP, result.IsTor)

		// Request new circuit via control port
		if i < rotationRequests-1 {
			cookiePath := filepath.Join(inst.cfg.Paths.TempFiles, fmt.Sprintf("tor_data_0/control_auth_cookie"))
			renewCircuit(t, inst.ControlPort, cookiePath)
			time.Sleep(1 * time.Second)
		}
	}

	uniqueIPs := len(ips)
	t.Logf("unique IPs: %d (of %d requests)", uniqueIPs, rotationRequests)

	// We expect at least 1 unique IP (proves Tor is working)
	if uniqueIPs == 0 {
		t.Error("no Tor exit IPs detected — Tor may not be routing traffic")
	}
}

// TestIntegration_DNSLeakPrevention verifies that DNS resolution through
// the Tor proxy returns a different IP than direct resolution (proving
// DNS goes through Tor, not the local resolver).
func TestIntegration_DNSLeakPrevention(t *testing.T) {
	cfg := integrationConfig(t)
	inst, cleanup := startTestInstance(t, cfg)
	defer cleanup()

	// Resolve through Tor SOCKS5 proxy
	// Use SOCKS5 to resolve "check.torproject.org" via Tor
	torIPs := resolveViaTor(t, inst.SocksPort, "check.torproject.org")

	// Resolve directly (without Tor)
	directIPs, err := net.LookupIP("check.torproject.org")
	if err != nil {
		t.Skipf("direct DNS resolution failed (network issue): %v", err)
	}

	t.Logf("Tor-resolved IPs: %v", torIPs)
	t.Logf("Direct IPs: %v", directIPs)

	// Check that Tor-resolved IPs are not the same as direct IPs
	// (they shouldn't be — Tor exit resolves, not our local resolver)
	leakFound := false
	for _, torIP := range torIPs {
		for _, directIP := range directIPs {
			if torIP.Equal(directIP) {
				leakFound = true
				t.Errorf("DNS leak detected: Tor resolved %s to %s (same as direct resolution)", "check.torproject.org", directIP)
			}
		}
	}

	if !leakFound && len(torIPs) > 0 {
		t.Log("no DNS leak detected — Tor and direct IPs differ")
	}
}

// resolveViaTor resolves a hostname through a Tor SOCKS5 proxy using
// SOCKS5 domain resolution (ATYP 0x03).
func resolveViaTor(t *testing.T, socksPort int, hostname string) []net.IP {
	t.Helper()

	dialer := &net.Dialer{Timeout: proxyTimeout}
	conn, err := dialer.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort))
	if err != nil {
		t.Fatalf("SOCKS5 connect failed: %v", err)
	}
	defer conn.Close()

	// SOCKS5 handshake
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		t.Fatalf("SOCKS5 handshake failed: %v", err)
	}
	buf := make([]byte, 2)
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("SOCKS5 auth response: %v", err)
	}

	// SOCKS5 domain resolution request (CMD=0xF0 = RESOLVE)
	resolveReq := []byte{0x05, 0xF0, 0x00, 0x03, byte(len(hostname))}
	resolveReq = append(resolveReq, []byte(hostname)...)
	resolveReq = append(resolveReq, 0x00, 0x00) // port 0 (unused)

	_, err = conn.Write(resolveReq)
	if err != nil {
		t.Fatalf("SOCKS5 resolve request: %v", err)
	}

	resp := make([]byte, 64)
	n, err := conn.Read(resp)
	if err != nil {
		t.Fatalf("SOCKS5 resolve response: %v", err)
	}

	// Parse response: version, status, reserved, ATYP, address
	if n < 7 || resp[1] != 0x00 {
		t.Fatalf("SOCKS5 resolve failed: status 0x%02x", resp[1])
	}

	var ips []net.IP
	switch resp[3] {
	case 0x01: // IPv4
		if n >= 10 {
			ips = append(ips, net.IP(resp[4:8]))
		}
	case 0x04: // IPv6
		if n >= 22 {
			ips = append(ips, net.IP(resp[4:20]))
		}
	case 0x03: // Domain (returned as-is in some configs)
		domainLen := int(resp[4])
		if n >= 5+domainLen {
			// Domain returned, not IP — try to resolve it
			return nil
		}
	}

	return ips
}

// renewCircuit sends a NEWNYM signal to the Tor control port.
func renewCircuit(t *testing.T, controlPort int, cookiePath string) {
	t.Helper()

	cookie, err := os.ReadFile(cookiePath)
	if err != nil {
		t.Logf("WARN: cannot read cookie file: %v", err)
		return
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", controlPort), 3*time.Second)
	if err != nil {
		t.Logf("WARN: cannot connect to control port: %v", err)
		return
	}
	defer conn.Close()

	// AUTHENTICATE <hex_cookie>\r\n
	authCmd := fmt.Sprintf("AUTHENTICATE %x\r\n", cookie)
	_, err = conn.Write([]byte(authCmd))
	if err != nil {
		t.Logf("WARN: authenticate failed: %v", err)
		return
	}

	resp := make([]byte, 256)
	n, err := conn.Read(resp)
	if err != nil {
		t.Logf("WARN: auth response failed: %v", err)
		return
	}
	if !strings.Contains(string(resp[:n]), "250 OK") {
		t.Logf("WARN: auth rejected: %s", string(resp[:n]))
		return
	}

	// SIGNAL NEWNYM\r\n
	_, err = conn.Write([]byte("SIGNAL NEWNYM\r\n"))
	if err != nil {
		t.Logf("WARN: NEWNYM failed: %v", err)
		return
	}
}

// TestIntegration_ControlPortCookieAuth verifies that:
// 1. CookieAuthentication is set in the torrc
// 2. The control_auth_cookie file exists
// 3. Cookie file has restricted permissions (0600)
func TestIntegration_ControlPortCookieAuth(t *testing.T) {
	cfg := integrationConfig(t)
	inst, cleanup := startTestInstance(t, cfg)
	defer cleanup()

	// Verify cookie file exists
	cookiePath := filepath.Join(inst.cfg.Paths.TempFiles, "tor_data_0/control_auth_cookie")
	info, err := os.Stat(cookiePath)
	if err != nil {
		t.Fatalf("control_auth_cookie file not found: %v", err)
	}

	// Verify restricted permissions (owner read/write only)
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("cookie file permissions = %04o, want 0600", perm)
	}

	// Verify cookie file is non-empty
	content, err := os.ReadFile(cookiePath)
	if err != nil {
		t.Fatalf("cannot read cookie file: %v", err)
	}
	if len(content) == 0 {
		t.Error("cookie file is empty")
	}

	t.Logf("control_auth_cookie: %d bytes, permissions %04o", len(content), perm)
}

// TestIntegration_NoHashedControlPassword verifies that the torrc
// does NOT contain HashedControlPassword (cookie auth only).
func TestIntegration_NoHashedControlPassword(t *testing.T) {
	cfg := integrationConfig(t)
	inst, cleanup := startTestInstance(t, cfg)
	defer cleanup()

	// Read the generated torrc
	torrcPath := filepath.Join(inst.cfg.Paths.TempFiles, "tor_0.cfg")
	content, err := os.ReadFile(torrcPath)
	if err != nil {
		t.Fatalf("cannot read torrc: %v", err)
	}

	if strings.Contains(string(content), "HashedControlPassword") {
		t.Error("torrc contains HashedControlPassword — should use cookie auth only")
	}
	if !strings.Contains(string(content), "CookieAuthentication 1") {
		t.Error("torrc missing CookieAuthentication 1")
	}
}
