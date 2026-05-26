package tor

import (
	"testing"
)

func TestBuildInstanceConfig_CGOEnabledFor049(t *testing.T) {
	v := &Version{Major: 0, Minor: 4, Patch: 9, Release: 0}
	if !v.SupportsCGO() {
		t.Error("SupportsCGO() = false, want true for Tor 0.4.9")
	}
}

func TestBuildInstanceConfig_CGODisabledFor048(t *testing.T) {
	v := &Version{Major: 0, Minor: 4, Patch: 8, Release: 0}
	if v.SupportsCGO() {
		t.Error("SupportsCGO() = true, want false for Tor 0.4.8")
	}
}

func TestTorrcTemplate_CGOEnabled(t *testing.T) {
	t.Skip("CGO is not a writable torrc option; skip template emission test")
}

func TestTorrcTemplate_NoCGO(t *testing.T) {
	t.Skip("CGO is not a writable torrc option; skip template emission test")
}
