package country

import (
	"math/rand/v2"
	"testing"
)

func TestSelectRandom_Basic(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}", "{GB}", "{NL}", "{SE}", "{CA}", "{AU}", "{JP}", "{BR}"}

	result, err := SelectRandom(accepted, nil, 3)
	if err != nil {
		t.Fatalf("SelectRandom() error = %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	seen := make(map[string]bool)
	for _, c := range result {
		if seen[c] {
			t.Errorf("duplicate country: %q", c)
		}
		seen[c] = true
	}
}

func TestSelectRandom_AllCountries(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}"}

	result, err := SelectRandom(accepted, nil, 3)
	if err != nil {
		t.Fatalf("SelectRandom() error = %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("len(result) = %d, want 3", len(result))
	}

	seen := make(map[string]bool)
	for _, c := range result {
		seen[c] = true
	}
	for _, c := range accepted {
		if !seen[c] {
			t.Errorf("missing country: %q", c)
		}
	}
}

func TestSelectRandom_ExceedsAvailable(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}"}

	_, err := SelectRandom(accepted, nil, 5)
	if err == nil {
		t.Error("SelectRandom() expected error when count > available, got nil")
	}
}

func TestSelectRandom_ExcludesBlacklisted(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}", "{GB}", "{NL}"}
	blacklisted := []string{"{DE}", "{FR}"}

	result, err := SelectRandom(accepted, blacklisted, 3)
	if err != nil {
		t.Fatalf("SelectRandom() error = %v", err)
	}

	for _, c := range result {
		if c == "{DE}" || c == "{FR}" {
			t.Errorf("blacklisted country %q in result", c)
		}
	}
}

func TestSelectRandom_SingleCountry(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}", "{GB}", "{NL}"}

	result, err := SelectRandom(accepted, nil, 1)
	if err != nil {
		t.Fatalf("SelectRandom() error = %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("len(result) = %d, want 1", len(result))
	}

	found := false
	for _, c := range accepted {
		if c == result[0] {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("result %q not in accepted list", result[0])
	}
}

func TestSelectRandom_DeterministicSeed(t *testing.T) {
	accepted := []string{"{US}", "{DE}", "{FR}", "{GB}", "{NL}", "{SE}", "{CA}", "{AU}", "{JP}", "{BR}"}

	r1 := rand.New(rand.NewPCG(42, 42))
	result1, err := selectRandom(accepted, nil, 3, r1.Shuffle)
	if err != nil {
		t.Fatalf("selectRandom() error = %v", err)
	}

	r2 := rand.New(rand.NewPCG(42, 42))
	result2, err := selectRandom(accepted, nil, 3, r2.Shuffle)
	if err != nil {
		t.Fatalf("selectRandom() error = %v", err)
	}

	if len(result1) != len(result2) {
		t.Fatalf("lengths differ: %d vs %d", len(result1), len(result2))
	}
	for i := range result1 {
		if result1[i] != result2[i] {
			t.Errorf("result[%d]: %q != %q", i, result1[i], result2[i])
		}
	}
}
