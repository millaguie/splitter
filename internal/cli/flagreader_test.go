package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestNewCobraFlagReader_NilFlags(t *testing.T) {
	reader := NewCobraFlagReader(nil)
	if reader == nil {
		t.Fatal("NewCobraFlagReader(nil) returned nil")
	}
	if reader.flags != nil {
		t.Error("expected nil flags")
	}
}

func TestCobraFlagReader_Changed(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("instances", 2, "instances")
	cmd.Flags().String("profile", "", "profile")

	reader := NewCobraFlagReader(cmd.Flags())

	if reader.Changed("instances") {
		t.Error("instances should not be changed before Set")
	}

	_ = cmd.Flags().Set("instances", "10")

	if !reader.Changed("instances") {
		t.Error("instances should be changed after Set")
	}
	if reader.Changed("profile") {
		t.Error("profile should not be changed")
	}
}

func TestCobraFlagReader_GetInt(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Int("count", 5, "count")

	reader := NewCobraFlagReader(cmd.Flags())

	val, err := reader.GetInt("count")
	if err != nil {
		t.Fatalf("GetInt(count) error = %v", err)
	}
	if val != 5 {
		t.Errorf("GetInt(count) = %d, want 5", val)
	}

	_ = cmd.Flags().Set("count", "20")
	val, err = reader.GetInt("count")
	if err != nil {
		t.Fatalf("GetInt(count) after set error = %v", err)
	}
	if val != 20 {
		t.Errorf("GetInt(count) = %d, want 20 after set", val)
	}
}

func TestCobraFlagReader_GetString(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("mode", "native", "mode")

	reader := NewCobraFlagReader(cmd.Flags())

	val, err := reader.GetString("mode")
	if err != nil {
		t.Fatalf("GetString(mode) error = %v", err)
	}
	if val != "native" {
		t.Errorf("GetString(mode) = %q, want native", val)
	}

	_ = cmd.Flags().Set("mode", "legacy")
	val, err = reader.GetString("mode")
	if err != nil {
		t.Fatalf("GetString(mode) after set error = %v", err)
	}
	if val != "legacy" {
		t.Errorf("GetString(mode) = %q, want legacy after set", val)
	}
}

func TestCobraFlagReader_GetBool(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("verbose", false, "verbose")

	reader := NewCobraFlagReader(cmd.Flags())

	val, err := reader.GetBool("verbose")
	if err != nil {
		t.Fatalf("GetBool(verbose) error = %v", err)
	}
	if val {
		t.Error("GetBool(verbose) = true, want false")
	}

	_ = cmd.Flags().Set("verbose", "true")
	val, err = reader.GetBool("verbose")
	if err != nil {
		t.Fatalf("GetBool(verbose) after set error = %v", err)
	}
	if !val {
		t.Error("GetBool(verbose) = false, want true after set")
	}
}

func TestCobraFlagReader_GetNonexistentFlag(t *testing.T) {
	cmd := &cobra.Command{}
	reader := NewCobraFlagReader(cmd.Flags())

	_, err := reader.GetInt("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent int flag")
	}

	_, err = reader.GetString("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent string flag")
	}

	_, err = reader.GetBool("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent bool flag")
	}
}

func TestBindFlags_RegistersAllFlags(t *testing.T) {
	cmd := &cobra.Command{}
	BindFlags(cmd)

	expectedFlags := []string{
		"instances", "countries", "relay-enforce",
		"profile", "proxy-mode", "bridge-type",
		"verbose", "log", "log-level", "auto-countries",
	}

	for _, name := range expectedFlags {
		f := cmd.PersistentFlags().Lookup(name)
		if f == nil {
			t.Errorf("flag %q not registered by BindFlags", name)
		}
	}
}

func TestBindFlags_DefaultValues(t *testing.T) {
	cmd := &cobra.Command{}
	BindFlags(cmd)

	tests := []struct {
		name string
		want string
	}{
		{"instances", "2"},
		{"countries", "6"},
		{"relay-enforce", "entry"},
		{"profile", ""},
		{"proxy-mode", "native"},
		{"bridge-type", "none"},
		{"verbose", "false"},
		{"log", "false"},
		{"log-level", "info"},
		{"auto-countries", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := cmd.PersistentFlags().Lookup(tt.name)
			if f == nil {
				t.Fatalf("flag %q not found", tt.name)
			}
			if f.DefValue != tt.want {
				t.Errorf("flag %q default = %q, want %q", tt.name, f.DefValue, tt.want)
			}
		})
	}
}

func TestBindFlags_ShortAliases(t *testing.T) {
	cmd := &cobra.Command{}
	BindFlags(cmd)

	shorthandTests := []struct {
		name      string
		shorthand string
	}{
		{"instances", "i"},
		{"countries", "c"},
		{"relay-enforce", "r"},
	}

	for _, tt := range shorthandTests {
		f := cmd.PersistentFlags().Lookup(tt.name)
		if f == nil {
			t.Fatalf("flag %q not found", tt.name)
		}
		if f.Shorthand != tt.shorthand {
			t.Errorf("flag %q shorthand = %q, want %q", tt.name, f.Shorthand, tt.shorthand)
		}
	}
}
