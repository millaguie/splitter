package proxy

import (
	"fmt"
	"strings"
	"testing"
	"text/template"
)

func TestPrivoxyTemplateFile_ExistsAndParses(t *testing.T) {
	tmpl, err := template.ParseFiles("../../templates/privoxy.cfg.gotmpl")
	if err != nil {
		t.Fatalf("template parse error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("parsed template is nil")
	}
}

func TestPrivoxyTemplateFile_RendersCorrectly(t *testing.T) {
	tmplStr := readFile(t, "../../templates/privoxy.cfg.gotmpl")

	data := privoxyConfigData{
		InstanceID: 0,
		ListenAddr: "127.0.0.1",
		Port:       8118,
		SocksPort:  9050,
	}

	result, err := RenderPrivoxyConfig(data, tmplStr)
	if err != nil {
		t.Fatalf("RenderPrivoxyConfig() error = %v", err)
	}

	privoxyTplContains(t, result, "listen-address 127.0.0.1:8118")
	privoxyTplContains(t, result, "forward-socks5t / 127.0.0.1:9050 .")
	privoxyTplContains(t, result, "toggle  1")
	privoxyTplContains(t, result, "buffer-limit 4096")
}

func TestPrivoxyTemplateFile_MultipleInstances(t *testing.T) {
	tmplStr := readFile(t, "../../templates/privoxy.cfg.gotmpl")

	tests := []struct {
		id        int
		addr      string
		port      int
		socksPort int
	}{
		{0, "127.0.0.1", 8118, 9050},
		{1, "127.0.0.1", 8119, 9051},
		{5, "0.0.0.0", 7005, 5005},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("instance_%d", tt.id), func(t *testing.T) {
			data := privoxyConfigData{
				InstanceID: tt.id,
				ListenAddr: tt.addr,
				Port:       tt.port,
				SocksPort:  tt.socksPort,
			}

			result, err := RenderPrivoxyConfig(data, tmplStr)
			if err != nil {
				t.Fatalf("RenderPrivoxyConfig() error = %v", err)
			}

			expectedListen := fmt.Sprintf("listen-address %s:%d", tt.addr, tt.port)
			if !strings.Contains(result, expectedListen) {
				t.Errorf("expected %q in output", expectedListen)
			}

			expectedForward := fmt.Sprintf("forward-socks5t / 127.0.0.1:%d .", tt.socksPort)
			if !strings.Contains(result, expectedForward) {
				t.Errorf("expected %q in output", expectedForward)
			}
		})
	}
}

func TestPrivoxyTemplateFile_SecuritySettings(t *testing.T) {
	tmplStr := readFile(t, "../../templates/privoxy.cfg.gotmpl")

	data := privoxyConfigData{
		InstanceID: 0,
		ListenAddr: "127.0.0.1",
		Port:       8118,
		SocksPort:  9050,
	}

	result, err := RenderPrivoxyConfig(data, tmplStr)
	if err != nil {
		t.Fatalf("RenderPrivoxyConfig() error = %v", err)
	}

	privoxyTplContains(t, result, "enable-remote-toggle 0")
	privoxyTplContains(t, result, "enable-edit-actions 0")
	privoxyTplContains(t, result, "enforce-blocks 1")
	privoxyTplContains(t, result, "logfile /dev/null")
}

func TestPrivoxyTemplateFile_PrivateNetworkForwards(t *testing.T) {
	tmplStr := readFile(t, "../../templates/privoxy.cfg.gotmpl")

	data := privoxyConfigData{
		InstanceID: 0,
		ListenAddr: "127.0.0.1",
		Port:       8118,
		SocksPort:  9050,
	}

	result, err := RenderPrivoxyConfig(data, tmplStr)
	if err != nil {
		t.Fatalf("RenderPrivoxyConfig() error = %v", err)
	}

	privoxyTplContains(t, result, "forward         10.0.0.0/8 .")
	privoxyTplContains(t, result, "forward         172.16.0.0/12 .")
	privoxyTplContains(t, result, "forward         192.168.0.0/16 .")
	privoxyTplContains(t, result, "forward         127.0.0.0/8 .")
}

func privoxyTplContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("expected output to contain %q\nfull output:\n%s", needle, haystack)
	}
}
