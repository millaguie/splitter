package cli

import (
	"reflect"
	"testing"
)

func TestPreprocessArgs_NoReFlag(t *testing.T) {
	args := []string{"splitter", "run", "-i", "5", "-c", "3"}
	result := PreprocessArgs(args)

	if len(result) != len(args) {
		t.Fatalf("PreprocessArgs length = %d, want %d", len(result), len(args))
	}
	for i := range args {
		if result[i] != args[i] {
			t.Errorf("result[%d] = %q, want %q", i, result[i], args[i])
		}
	}
}

func TestPreprocessArgs_ReFlag(t *testing.T) {
	args := []string{"splitter", "-re", "exit"}
	result := PreprocessArgs(args)
	expected := []string{"splitter", "--relay-enforce", "exit"}

	if len(result) != len(expected) {
		t.Fatalf("length = %d, want %d", len(result), len(expected))
	}
	for i := range expected {
		if result[i] != expected[i] {
			t.Errorf("result[%d] = %q, want %q", i, result[i], expected[i])
		}
	}
}

func TestPreprocessArgs_MultipleReFlags(t *testing.T) {
	args := []string{"-re", "exit", "-re", "entry"}
	result := PreprocessArgs(args)
	expected := []string{"--relay-enforce", "exit", "--relay-enforce", "entry"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("PreprocessArgs() = %v, want %v", result, expected)
	}
}

func TestPreprocessArgs_EmptySlice(t *testing.T) {
	result := PreprocessArgs([]string{})

	if result == nil {
		t.Error("PreprocessArgs(empty) returned nil, want empty slice")
	}
	if len(result) != 0 {
		t.Errorf("PreprocessArgs(empty) returned %d elements, want 0", len(result))
	}
}

func TestPreprocessArgs_NilSlice(t *testing.T) {
	result := PreprocessArgs(nil)

	if result == nil {
		t.Error("PreprocessArgs(nil) returned nil, want empty slice")
	}
	if len(result) != 0 {
		t.Errorf("PreprocessArgs(nil) returned %d elements, want 0", len(result))
	}
}

func TestPreprocessArgs_MixedFlags(t *testing.T) {
	args := []string{"splitter", "-re", "exit", "-i", "3", "--profile", "stealth"}
	result := PreprocessArgs(args)
	expected := []string{"splitter", "--relay-enforce", "exit", "-i", "3", "--profile", "stealth"}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("PreprocessArgs() = %v, want %v", result, expected)
	}
}

func TestPreprocessArgs_PreservesOtherFlags(t *testing.T) {
	args := []string{"--profile", "stealth", "--proxy-mode", "legacy"}
	result := PreprocessArgs(args)

	if len(result) != len(args) {
		t.Fatalf("length = %d, want %d", len(result), len(args))
	}
	for i := range args {
		if result[i] != args[i] {
			t.Errorf("result[%d] = %q, want %q", i, result[i], args[i])
		}
	}
}
