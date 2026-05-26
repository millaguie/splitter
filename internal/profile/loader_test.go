package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	yaml := `
stealth:
  description: "test stealth"
  instances:
    per_country: 3
  tor:
    conflux_enabled: true
`
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	profiles, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	p, ok := profiles["stealth"]
	if !ok {
		t.Fatal("stealth profile not found")
	}
	if p.Description != "test stealth" {
		t.Errorf("Description = %q, want %q", p.Description, "test stealth")
	}
	if p.Tor.ConfluxEnabled == nil || !*p.Tor.ConfluxEnabled {
		t.Error("conflux_enabled should be true")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/profiles.yaml")
	if err == nil {
		t.Error("Load() expected error for missing file, got nil")
	}
}

func TestLoad_MalformedYAML(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte("stealth: [broken yaml\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("Load() expected error for malformed YAML, got nil")
	}
}

func TestLoad_AllProfilesFromProject(t *testing.T) {
	profiles, err := Load(filepath.Join("..", "..", "configs", "profiles.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, name := range ValidProfiles {
		p, ok := profiles[name]
		if !ok {
			t.Errorf("profile %q not found", name)
			continue
		}
		if p.Description == "" {
			t.Errorf("profile %q has empty description", name)
		}
		if p.Instances.PerCountry == nil || *p.Instances.PerCountry <= 0 {
			t.Errorf("profile %q has invalid per_country", name)
		}
		if p.Instances.Countries == nil || *p.Instances.Countries <= 0 {
			t.Errorf("profile %q has invalid countries", name)
		}
		if p.Relay.Enforce == nil {
			t.Errorf("profile %q has nil relay.enforce", name)
		}
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "profiles.yaml")
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	profiles, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(profiles) != 0 {
		t.Errorf("expected 0 profiles from empty file, got %d", len(profiles))
	}
}
