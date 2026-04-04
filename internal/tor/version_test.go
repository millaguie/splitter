package tor

import (
	"testing"
)

func TestParseVersionOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    *Version
		wantErr bool
	}{
		{
			name:   "standard 3-component format",
			output: "Tor version 0.4.8.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 8, Release: 0},
		},
		{
			name:   "4-component format 0.4.8.17",
			output: "Tor version 0.4.8.17.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 8, Release: 17},
		},
		{
			name:   "with extra text",
			output: "Tor version 0.4.7.13 (git-1234abcd).\nTor is running on Linux.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 7, Release: 13},
		},
		{
			name:   "newer version 0.4.9.1",
			output: "Tor version 0.4.9.1.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 9, Release: 1},
		},
		{
			name:   "0.4.7.0",
			output: "Tor version 0.4.7.0.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 7, Release: 0},
		},
		{
			name:   "0.4.9.5",
			output: "Tor version 0.4.9.5.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 9, Release: 5},
		},
		{
			name:    "no version",
			output:  "Some other program version 1.2.3\n",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:   "version in middle of line",
			output: "Starting Tor version 0.4.8.0 running on x86_64\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 8, Release: 0},
		},
		{
			name:   "version 0.4.8.16 no PQ",
			output: "Tor version 0.4.8.16.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 8, Release: 16},
		},
		{
			name:   "version 0.4.8.17 has PQ",
			output: "Tor version 0.4.8.17.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 8, Release: 17},
		},
		{
			name:   "version 0.4.9.0 has PQ (0.4.9 > 0.4.8.17)",
			output: "Tor version 0.4.9.0.\n",
			want:   &Version{Major: 0, Minor: 4, Patch: 9, Release: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DetectVersionFromOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectVersionFromOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Major != tt.want.Major || got.Minor != tt.want.Minor ||
				got.Patch != tt.want.Patch || got.Release != tt.want.Release {
				t.Errorf("DetectVersionFromOutput() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestVersion_FeatureDetection(t *testing.T) {
	tests := []struct {
		name              string
		version           *Version
		conflux           bool
		httpTunnel        bool
		congestionControl bool
		cgo               bool
		postQuantum       bool
		happyFamilies     bool
		tls13             bool
		sandbox           bool
	}{
		{"0.4.6.99", &Version{0, 4, 6, 99}, false, false, false, false, false, false, false, false},
		{"0.4.7.0", &Version{0, 4, 7, 0}, false, false, true, false, false, false, false, true},
		{"0.4.7.9", &Version{0, 4, 7, 9}, false, false, true, false, false, false, false, true},
		{"0.4.8.0", &Version{0, 4, 8, 0}, true, true, true, false, false, false, false, true},
		{"0.4.8.10", &Version{0, 4, 8, 10}, true, true, true, false, false, false, false, true},
		{"0.4.8.16", &Version{0, 4, 8, 16}, true, true, true, false, false, false, false, true},
		{"0.4.8.17 PQ boundary", &Version{0, 4, 8, 17}, true, true, true, false, true, false, false, true},
		{"0.4.8.22", &Version{0, 4, 8, 22}, true, true, true, false, true, false, false, true},
		{"0.4.9.0", &Version{0, 4, 9, 0}, true, true, true, true, true, true, true, true},
		{"0.4.9.5", &Version{0, 4, 9, 5}, true, true, true, true, true, true, true, true},
		{"0.5.0.0", &Version{0, 5, 0, 0}, true, true, true, true, true, true, true, true},
		{"1.0.0.0", &Version{1, 0, 0, 0}, true, true, true, true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.SupportsConflux(); got != tt.conflux {
				t.Errorf("SupportsConflux() = %v, want %v", got, tt.conflux)
			}
			if got := tt.version.SupportsHTTPTunnel(); got != tt.httpTunnel {
				t.Errorf("SupportsHTTPTunnel() = %v, want %v", got, tt.httpTunnel)
			}
			if got := tt.version.SupportsCongestionControl(); got != tt.congestionControl {
				t.Errorf("SupportsCongestionControl() = %v, want %v", got, tt.congestionControl)
			}
			if got := tt.version.SupportsCGO(); got != tt.cgo {
				t.Errorf("SupportsCGO() = %v, want %v", got, tt.cgo)
			}
			if got := tt.version.SupportsPostQuantum(); got != tt.postQuantum {
				t.Errorf("SupportsPostQuantum() = %v, want %v", got, tt.postQuantum)
			}
			if got := tt.version.SupportsHappyFamilies(); got != tt.happyFamilies {
				t.Errorf("SupportsHappyFamilies() = %v, want %v", got, tt.happyFamilies)
			}
			if got := tt.version.SupportsTLS13(); got != tt.tls13 {
				t.Errorf("SupportsTLS13() = %v, want %v", got, tt.tls13)
			}
			if got := tt.version.SupportsSandbox(); got != tt.sandbox {
				t.Errorf("SupportsSandbox() = %v, want %v", got, tt.sandbox)
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 17}
	if got := v.String(); got != "0.4.8.17" {
		t.Errorf("String() = %q, want %q", got, "0.4.8.17")
	}
}

func TestVersion_StringZeroRelease(t *testing.T) {
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	if got := v.String(); got != "0.4.8.0" {
		t.Errorf("String() = %q, want %q", got, "0.4.8.0")
	}
}

func TestBackoffDuration(t *testing.T) {
	tests := []struct {
		failures int
		wantSec  float64
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 4},
		{4, 8},
		{5, 16},
		{6, 30},
		{10, 30},
		{100, 30},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := backoffDuration(tt.failures)
			if got.Seconds() != tt.wantSec {
				t.Errorf("backoffDuration(%d) = %v, want %vs", tt.failures, got, tt.wantSec)
			}
		})
	}
}

func TestDetectTLSInfoFromOutput(t *testing.T) {
	tests := []struct {
		name        string
		output      string
		wantVersion string
		wantPQ      bool
	}{
		{
			name:        "OpenSSL 3.5.0 PQ capable",
			output:      "Tor version 0.4.8.17.\nTor is running on Linux with Libevent 2.1.12-stable, OpenSSL 3.5.0, Zlib 1.2.13, Liblzma 5.4.1, and Libzstd 1.5.4.\n",
			wantVersion: "OpenSSL 3.5.0",
			wantPQ:      true,
		},
		{
			name:        "OpenSSL 3.0.0 not PQ capable",
			output:      "Tor version 0.4.8.17.\nTor is running on Linux with Libevent 2.1.12-stable, OpenSSL 3.0.0, Zlib 1.2.13.\n",
			wantVersion: "OpenSSL 3.0.0",
			wantPQ:      false,
		},
		{
			name:        "OpenSSL 1.1.1 not PQ capable",
			output:      "Tor version 0.4.8.10.\nTor is running on Linux with OpenSSL 1.1.1w.\n",
			wantVersion: "OpenSSL 1.1.1",
			wantPQ:      false,
		},
		{
			name:        "OpenSSL 3.6.0 PQ capable",
			output:      "Tor version 0.4.9.5.\nTor is running on Linux with OpenSSL 3.6.0.\n",
			wantVersion: "OpenSSL 3.6.0",
			wantPQ:      true,
		},
		{
			name:        "no OpenSSL info",
			output:      "Tor version 0.4.7.13.\n",
			wantVersion: "",
			wantPQ:      false,
		},
		{
			name:        "LibreSSL not PQ capable",
			output:      "Tor version 0.4.8.17.\nTor is running on Linux with LibreSSL 3.8.0.\n",
			wantVersion: "",
			wantPQ:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := DetectTLSInfoFromOutput(tt.output)
			if info.OpenSSLVersion != tt.wantVersion {
				t.Errorf("OpenSSLVersion = %q, want %q", info.OpenSSLVersion, tt.wantVersion)
			}
			if info.PostQuantumOK != tt.wantPQ {
				t.Errorf("PostQuantumOK = %v, want %v", info.PostQuantumOK, tt.wantPQ)
			}
		})
	}
}
