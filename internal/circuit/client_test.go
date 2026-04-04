package circuit

import (
	"strings"
	"testing"
)

func TestClient_AuthenticateCookieHex(t *testing.T) {
	cookie := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xab, 0xcd, 0xef}
	cmd := BuildAuthCommand(cookie)

	expected := "AUTHENTICATE 0123456789abcdef\r\n"
	if cmd != expected {
		t.Errorf("BuildAuthCommand() = %q, want %q", cmd, expected)
	}
}

func TestClient_AuthenticateCookieHexEmpty(t *testing.T) {
	cmd := BuildAuthCommand([]byte{})

	expected := "AUTHENTICATE \r\n"
	if cmd != expected {
		t.Errorf("BuildAuthCommand(empty) = %q, want %q", cmd, expected)
	}
}

func TestClient_ParseResponse_250(t *testing.T) {
	code, msg, err := ParseResponse("250 OK")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want %q", code, "250")
	}
	if msg != "OK" {
		t.Errorf("message = %q, want %q", msg, "OK")
	}
}

func TestClient_ParseResponse_250WithCRLF(t *testing.T) {
	code, msg, err := ParseResponse("250 OK\r\n")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want %q", code, "250")
	}
	if msg != "OK" {
		t.Errorf("message = %q, want %q", msg, "OK")
	}
}

func TestClient_ParseResponse_MidReply(t *testing.T) {
	code, msg, err := ParseResponse("250-PROTOCOLINFO")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want %q", code, "250")
	}
	if msg != "-PROTOCOLINFO" {
		t.Errorf("message = %q, want %q", msg, "-PROTOCOLINFO")
	}
}

func TestClient_ParseResponse_515BadAuth(t *testing.T) {
	code, msg, err := ParseResponse("515 Bad authentication")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "515" {
		t.Errorf("code = %q, want %q", code, "515")
	}
	if msg != "Bad authentication" {
		t.Errorf("message = %q, want %q", msg, "Bad authentication")
	}
}

func TestClient_ParseResponse_650StatusEvent(t *testing.T) {
	code, msg, err := ParseResponse("650 STREAM 1234 NEW 0 example.com:443")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "650" {
		t.Errorf("code = %q, want %q", code, "650")
	}
	if !strings.Contains(msg, "STREAM") {
		t.Errorf("message should contain 'STREAM', got %q", msg)
	}
}

func TestClient_ParseResponse_ExactlyThreeChars(t *testing.T) {
	_, _, err := ParseResponse("250")
	if err == nil {
		t.Fatal("ParseResponse should reject 3-char input with no space separator")
	}
}

func TestClient_ParseResponse_ThreeCharsPlusSpace(t *testing.T) {
	code, msg, err := ParseResponse("250 ")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want %q", code, "250")
	}
	if msg != "" {
		t.Errorf("message = %q, want empty (space trimmed)", msg)
	}
}

func TestClient_ParseResponse_250WithExtra(t *testing.T) {
	code, msg, err := ParseResponse("250-SOME MULTILINE")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want %q", code, "250")
	}
	if msg != "-SOME MULTILINE" {
		t.Errorf("message = %q, want %q", msg, "-SOME MULTILINE")
	}
}

func TestClient_ParseResponse_Error(t *testing.T) {
	code, msg, err := ParseResponse("515 Bad authentication")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "515" {
		t.Errorf("code = %q, want %q", code, "515")
	}
	if msg != "Bad authentication" {
		t.Errorf("message = %q, want %q", msg, "Bad authentication")
	}
}

func TestClient_ParseResponse_TooShort(t *testing.T) {
	_, _, err := ParseResponse("25")
	if err == nil {
		t.Error("expected error for short response, got nil")
	}
}

func TestClient_ParseResponse_Empty(t *testing.T) {
	_, _, err := ParseResponse("")
	if err == nil {
		t.Error("expected error for empty response, got nil")
	}
}

func TestClient_SignalNewnym(t *testing.T) {
	cmd := BuildNewnymCommand()

	expected := "SIGNAL NEWNYM\r\n"
	if cmd != expected {
		t.Errorf("BuildNewnymCommand() = %q, want %q", cmd, expected)
	}
}

func TestClient_CloseWhenNil(t *testing.T) {
	c := NewClient("127.0.0.1:9051", "/tmp/cookie")
	if err := c.Close(); err != nil {
		t.Errorf("Close() on nil conn error = %v, want nil", err)
	}
}

func TestNewClient_Fields(t *testing.T) {
	c := NewClient("127.0.0.1:9052", "/tmp/test_cookie")
	if c.addr != "127.0.0.1:9052" {
		t.Errorf("addr = %q, want %q", c.addr, "127.0.0.1:9052")
	}
	if c.cookiePath != "/tmp/test_cookie" {
		t.Errorf("cookiePath = %q, want %q", c.cookiePath, "/tmp/test_cookie")
	}
	if c.conn != nil {
		t.Error("conn should be nil on creation")
	}
}

func TestBuildAuthCommand_FullCookie(t *testing.T) {
	cookie := []byte{0x00, 0xff, 0xab, 0xcd, 0x12, 0x34, 0x56, 0x78,
		0x9a, 0xbc, 0xde, 0xf0, 0x11, 0x22, 0x33, 0x44,
		0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc,
		0xdd, 0xee, 0xff, 0x00, 0x11, 0x22, 0x33, 0x44}
	cmd := BuildAuthCommand(cookie)

	if !strings.HasPrefix(cmd, "AUTHENTICATE ") {
		t.Errorf("command should start with AUTHENTICATE, got %q", cmd[:20])
	}
	if !strings.HasSuffix(cmd, "\r\n") {
		t.Error("command should end with CRLF")
	}

	hexPart := strings.TrimSuffix(strings.TrimPrefix(cmd, "AUTHENTICATE "), "\r\n")
	if len(hexPart) != 64 {
		t.Errorf("hex cookie length = %d, want 64 (32 bytes hex-encoded)", len(hexPart))
	}
}

func TestParseResponse_NumericCodes(t *testing.T) {
	tests := []struct {
		input string
		code  string
	}{
		{"250 OK", "250"},
		{"510 Command not recognized", "510"},
		{"515 Bad authentication", "515"},
		{"552 Unrecognized info", "552"},
		{"650 STREAM", "650"},
	}

	for _, tt := range tests {
		t.Run(tt.input[:3], func(t *testing.T) {
			code, _, err := ParseResponse(tt.input)
			if err != nil {
				t.Fatalf("ParseResponse() error = %v", err)
			}
			if code != tt.code {
				t.Errorf("code = %q, want %q", code, tt.code)
			}
		})
	}
}
