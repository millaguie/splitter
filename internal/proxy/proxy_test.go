package proxy

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Privoxy.BinaryPath = "/usr/sbin/privoxy"
	cfg.Privoxy.Listen = "127.0.0.1"
	cfg.Privoxy.StartPort = 6999
	cfg.Privoxy.Timeout = 35
	cfg.Privoxy.ConfigFilePrefix = t.TempDir() + "/privoxy_splitter_config_"
	cfg.Paths.TempFiles = t.TempDir()
	return cfg
}

func TestNewProxy_NativeMode(t *testing.T) {
	p := NewProxy(ModeNative, nil, nil)
	if _, ok := p.(*NativeProxy); !ok {
		t.Errorf("NewProxy(native) = %T, want *NativeProxy", p)
	}
}

func TestNewProxy_LegacyMode(t *testing.T) {
	cfg := testConfig(t)
	p := NewProxy(ModeLegacy, cfg, process.NewManager(""))
	if _, ok := p.(*LegacyProxy); !ok {
		t.Errorf("NewProxy(legacy) = %T, want *LegacyProxy", p)
	}
}

func TestNativeProxy_Setup(t *testing.T) {
	p := &NativeProxy{}
	instances := []Instance{
		{ID: 0, SocksPort: 4999, HTTPPort: 5199},
		{ID: 1, SocksPort: 5000, HTTPPort: 5200},
	}

	ports, err := p.Setup(context.Background(), instances)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	if len(ports) != 2 {
		t.Fatalf("Setup() returned %d ports, want 2", len(ports))
	}
	if ports[0] != 5199 {
		t.Errorf("ports[0] = %d, want 5199", ports[0])
	}
	if ports[1] != 5200 {
		t.Errorf("ports[1] = %d, want 5200", ports[1])
	}
}

func TestNativeProxy_Setup_NoHTTPTunnel(t *testing.T) {
	p := &NativeProxy{}
	instances := []Instance{
		{ID: 0, SocksPort: 4999, HTTPPort: 0},
	}

	_, err := p.Setup(context.Background(), instances)
	if err == nil {
		t.Fatal("Setup() expected error when HTTPPort == 0, got nil")
	}
	if !strings.Contains(err.Error(), "legacy") {
		t.Errorf("error should suggest legacy mode, got: %v", err)
	}
}

func TestNativeProxy_Setup_MixedInstances(t *testing.T) {
	p := &NativeProxy{}
	instances := []Instance{
		{ID: 0, SocksPort: 4999, HTTPPort: 5199},
		{ID: 1, SocksPort: 5000, HTTPPort: 0},
	}

	_, err := p.Setup(context.Background(), instances)
	if err == nil {
		t.Fatal("Setup() expected error when some instances have HTTPPort == 0")
	}
}

func TestNativeProxy_StartStop(t *testing.T) {
	p := &NativeProxy{}

	if err := p.Start(context.Background()); err != nil {
		t.Errorf("Start() error = %v, want nil", err)
	}
	if err := p.Stop(context.Background()); err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}
}

func TestNativeProxy_Mode(t *testing.T) {
	p := &NativeProxy{}
	if p.Mode() != ModeNative {
		t.Errorf("Mode() = %q, want %q", p.Mode(), ModeNative)
	}
}

func TestLegacyProxy_Setup(t *testing.T) {
	cfg := testConfig(t)
	p := &LegacyProxy{cfg: cfg}

	instances := []Instance{
		{ID: 0, SocksPort: 4999, HTTPPort: 5199},
		{ID: 1, SocksPort: 5000, HTTPPort: 5200},
	}

	ports, err := p.Setup(context.Background(), instances)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}

	if len(ports) != 2 {
		t.Fatalf("Setup() returned %d ports, want 2", len(ports))
	}
	if ports[0] != 6999 {
		t.Errorf("ports[0] = %d, want 6999", ports[0])
	}
	if ports[1] != 7000 {
		t.Errorf("ports[1] = %d, want 7000", ports[1])
	}

	for _, id := range []int{0, 1} {
		configPath := cfg.Privoxy.ConfigFilePrefix + "0.cfg"
		if id == 1 {
			configPath = cfg.Privoxy.ConfigFilePrefix + "1.cfg"
		}
		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Errorf("config file %s: %v", configPath, err)
			continue
		}
		content := string(data)
		if !strings.Contains(content, "listen-address") {
			t.Errorf("config for instance %d missing listen-address", id)
		}
		if !strings.Contains(content, "forward-socks5t") {
			t.Errorf("config for instance %d missing forward-socks5t", id)
		}
	}
}

func TestLegacyProxy_Mode(t *testing.T) {
	p := &LegacyProxy{}
	if p.Mode() != ModeLegacy {
		t.Errorf("Mode() = %q, want %q", p.Mode(), ModeLegacy)
	}
}

func TestLegacyProxy_Start_NoPorts(t *testing.T) {
	cfg := testConfig(t)
	p := &LegacyProxy{cfg: cfg}

	if err := p.Start(context.Background()); err != nil {
		t.Errorf("Start() with no ports error = %v, want nil", err)
	}
}

func TestLegacyProxy_Stop_NoProcs(t *testing.T) {
	cfg := testConfig(t)
	p := &LegacyProxy{cfg: cfg}

	if err := p.Stop(context.Background()); err != nil {
		t.Errorf("Stop() with no procs error = %v, want nil", err)
	}
}

func TestParseMode(t *testing.T) {
	tests := []struct {
		input   string
		want    Mode
		wantErr bool
	}{
		{"native", ModeNative, false},
		{"legacy", ModeLegacy, false},
		{"invalid", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseMode(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("ParseMode() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseMode() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("ParseMode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPrivoxyTemplate(t *testing.T) {
	tmpl := readFile(t, "../../templates/privoxy.cfg.gotmpl")

	data := privoxyConfigData{
		InstanceID: 3,
		ListenAddr: "127.0.0.1",
		Port:       7002,
		SocksPort:  5002,
	}

	result, err := RenderPrivoxyConfig(data, tmpl)
	if err != nil {
		t.Fatalf("RenderPrivoxyConfig() error = %v", err)
	}

	assertContains(t, result, "Instance 3")
	assertContains(t, result, "listen-address 127.0.0.1:7002")
	assertContains(t, result, "forward-socks5t / 127.0.0.1:5002 .")
	assertContains(t, result, "toggle  1")
	assertContains(t, result, "enable-remote-toggle 0")
	assertContains(t, result, "enable-edit-actions 0")
	assertContains(t, result, "enforce-blocks 1")
	assertContains(t, result, "logfile /dev/null")
	assertContains(t, result, "buffer-limit 4096")
}

func TestPrivoxyTemplate_DefaultInline(t *testing.T) {
	data := privoxyConfigData{
		InstanceID: 0,
		ListenAddr: "0.0.0.0",
		Port:       6999,
		SocksPort:  4999,
	}

	result, err := RenderPrivoxyConfig(data, defaultPrivoxyTmpl)
	if err != nil {
		t.Fatalf("RenderPrivoxyConfig() error = %v", err)
	}

	assertContains(t, result, "listen-address 0.0.0.0:6999")
	assertContains(t, result, "forward-socks5t / 127.0.0.1:4999 .")
	assertContains(t, result, "forward         10.0.0.0/8 .")
	assertContains(t, result, "forward         172.16.0.0/12 .")
	assertContains(t, result, "forward         192.168.0.0/16 .")
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("readFile %s: %v", path, err)
	}
	return string(data)
}

func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", needle, haystack)
	}
}
