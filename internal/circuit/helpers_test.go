package circuit

import (
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

func TestBuildAuthCommand_SingleByteCookie(t *testing.T) {
	cookie := []byte{0xaa}
	cmd := BuildAuthCommand(cookie)
	expected := "AUTHENTICATE " + hex.EncodeToString(cookie) + "\r\n"
	if cmd != expected {
		t.Errorf("BuildAuthCommand(1 byte) = %q, want %q", cmd, expected)
	}
}

func TestBuildAuthCommand_16ByteCookie(t *testing.T) {
	cookie := make([]byte, 16)
	for i := range cookie {
		cookie[i] = byte(i)
	}
	cmd := BuildAuthCommand(cookie)

	hexPart := strings.TrimSuffix(strings.TrimPrefix(cmd, "AUTHENTICATE "), "\r\n")
	if len(hexPart) != 32 {
		t.Errorf("hex cookie length = %d, want 32 (16 bytes)", len(hexPart))
	}
}

func TestBuildAuthCommand_AllZeroCookie(t *testing.T) {
	cookie := make([]byte, 32)
	cmd := BuildAuthCommand(cookie)
	if !strings.HasPrefix(cmd, "AUTHENTICATE ") {
		t.Errorf("command should start with AUTHENTICATE, got %q", cmd[:20])
	}
	if !strings.HasSuffix(cmd, "\r\n") {
		t.Error("command should end with CRLF")
	}
	hexPart := strings.TrimSuffix(strings.TrimPrefix(cmd, "AUTHENTICATE "), "\r\n")
	expected := strings.Repeat("00", 32)
	if hexPart != expected {
		t.Errorf("hex part = %q, want %q", hexPart, expected)
	}
}

func TestBuildNewnymCommand_Immutable(t *testing.T) {
	cmd1 := BuildNewnymCommand()
	cmd2 := BuildNewnymCommand()
	if cmd1 != cmd2 {
		t.Errorf("BuildNewnymCommand() returned different values: %q vs %q", cmd1, cmd2)
	}
}

func TestParseResponse_WithOnlyCode(t *testing.T) {
	code, msg, err := ParseResponse("250")
	if err == nil {
		t.Fatal("ParseResponse should reject 3-char input with no separator")
	}
	if code != "" || msg != "" {
		t.Errorf("expected empty code/msg on error, got code=%q msg=%q", code, msg)
	}
}

func TestParseResponse_CodeOnlyWithSpace(t *testing.T) {
	code, msg, err := ParseResponse("250 ")
	if err != nil {
		t.Fatalf("ParseResponse() error = %v", err)
	}
	if code != "250" {
		t.Errorf("code = %q, want 250", code)
	}
	if msg != "" {
		t.Errorf("message = %q, want empty", msg)
	}
}

func TestParseResponse_TwoCharInput(t *testing.T) {
	_, _, err := ParseResponse("25")
	if err == nil {
		t.Error("expected error for 2-char input")
	}
}

func TestParseResponse_OneCharInput(t *testing.T) {
	_, _, err := ParseResponse("2")
	if err == nil {
		t.Error("expected error for 1-char input")
	}
}

func TestRandomInterval_LargeRange(t *testing.T) {
	min := 1 * time.Second
	max := 60 * time.Second
	for i := 0; i < 100; i++ {
		got := randomInterval(min, max)
		if got < min || got > max {
			t.Errorf("randomInterval(1s, 60s) = %v, out of range", got)
		}
	}
}

func TestRandomInterval_OneNanosecondDelta(t *testing.T) {
	min := 10 * time.Second
	max := 10*time.Second + 1
	got := randomInterval(min, max)
	if got < min || got > max {
		t.Errorf("randomInterval(10s, 10s+1ns) = %v, out of range", got)
	}
}
