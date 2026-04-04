package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReputationChecker_Check_Success(t *testing.T) {
	response := onionooDetailsResponse{
		Relays: []onionooDetailsRelay{
			{
				Fingerprint:       "AABB1122",
				Country:           "de",
				ObservedBandwidth: 5_000_000,
				Flags:             []string{"Exit", "Fast", "Stable", "Running"},
				FirstSeen:         "2025-01-15",
				LastSeen:          "2026-04-02",
			},
			{
				Fingerprint:       "CCDD3344",
				Country:           "de",
				ObservedBandwidth: 500_000,
				Flags:             []string{"Exit", "Running"},
				FirstSeen:         time.Now().Format("2006-01-02"),
				LastSeen:          time.Now().Format("2006-01-02"),
			},
		},
	}

	srv := httptest.NewServer(reputationJSONHandler(t, response))
	defer srv.Close()

	tmpDir := t.TempDir()
	rc := NewReputationChecker(filepath.Join(tmpDir, "reputation_cache.json"))
	rc.SetAPIURL(srv.URL)

	reps, err := rc.Check(context.Background(), "de")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if len(reps) != 2 {
		t.Fatalf("len(reps) = %d, want 2", len(reps))
	}

	if reps[0].Country != "DE" {
		t.Errorf("Country = %q, want DE", reps[0].Country)
	}
	if reps[0].Fingerprint != "AABB1122" {
		t.Errorf("Fingerprint = %q, want AABB1122", reps[0].Fingerprint)
	}
	if reps[0].UptimeDays <= 0 {
		t.Errorf("UptimeDays = %d, want > 0", reps[0].UptimeDays)
	}
	if reps[1].IsNew != true {
		t.Errorf("IsNew = %v, want true for relay with first_seen today", reps[1].IsNew)
	}
}

func TestReputationChecker_Check_WritesCache(t *testing.T) {
	response := onionooDetailsResponse{
		Relays: []onionooDetailsRelay{
			{
				Fingerprint:       "AABB1122",
				Country:           "us",
				ObservedBandwidth: 2_000_000,
				Flags:             []string{"Exit", "Fast"},
				FirstSeen:         "2025-06-01",
				LastSeen:          "2026-04-02",
			},
		},
	}

	srv := httptest.NewServer(reputationJSONHandler(t, response))
	defer srv.Close()

	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")
	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	_, err := rc.Check(context.Background(), "us")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	var cache reputationCache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("cache unmarshal: %v", err)
	}
	if cache.FetchedAt.IsZero() {
		t.Error("cached fetched_at is zero")
	}
	if _, ok := cache.Entries["US"]; !ok {
		t.Error("cache missing US entry")
	}
}

func TestReputationChecker_Check_UsesCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	cached := reputationCache{
		FetchedAt: time.Now().UTC(),
		Entries: map[string][]ExitReputation{
			"DE": {
				{Fingerprint: "CACHED01", Country: "DE", Score: 0.8},
			},
		},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("API should not be called when cache is fresh")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	reps, err := rc.Check(context.Background(), "de")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(reps) != 1 || reps[0].Fingerprint != "CACHED01" {
		t.Errorf("reps = %v, want cached entry", reps)
	}
}

func TestReputationChecker_Check_TTLExpired(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	cached := reputationCache{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Entries: map[string][]ExitReputation{
			"DE": {
				{Fingerprint: "STALE01", Country: "DE", Score: 0.5},
			},
		},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	response := onionooDetailsResponse{
		Relays: []onionooDetailsRelay{
			{
				Fingerprint:       "FRESH01",
				Country:           "de",
				ObservedBandwidth: 3_000_000,
				Flags:             []string{"Exit", "Fast", "Stable"},
				FirstSeen:         "2024-01-01",
				LastSeen:          "2026-04-02",
			},
		},
	}

	srv := httptest.NewServer(reputationJSONHandler(t, response))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	reps, err := rc.Check(context.Background(), "de")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(reps) != 1 || reps[0].Fingerprint != "FRESH01" {
		t.Errorf("reps = %v, want fresh entry", reps)
	}
}

func TestReputationChecker_Check_StaleCacheOnFetchError(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	cached := reputationCache{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Entries: map[string][]ExitReputation{
			"FR": {
				{Fingerprint: "STALE_FR", Country: "FR", Score: 0.6},
			},
		},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	reps, err := rc.Check(context.Background(), "fr")
	if err != nil {
		t.Fatalf("Check() error = %v, want stale cache fallback", err)
	}
	if len(reps) != 1 || reps[0].Fingerprint != "STALE_FR" {
		t.Errorf("reps = %v, want stale cache entry", reps)
	}
}

func TestReputationChecker_Check_NoCacheFetchError(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	_, err := rc.Check(context.Background(), "xx")
	if err == nil {
		t.Error("Check() expected error when fetch fails with no cache")
	}
}

func TestReputationChecker_Check_EmptyResponseNoCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	response := onionooDetailsResponse{Relays: []onionooDetailsRelay{}}
	srv := httptest.NewServer(reputationJSONHandler(t, response))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	_, err := rc.Check(context.Background(), "zz")
	if err == nil {
		t.Error("Check() expected error for empty API response with no cache")
	}
}

func TestReputationChecker_Check_EmptyResponseWithStaleCache(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	cached := reputationCache{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Entries: map[string][]ExitReputation{
			"ZZ": {
				{Fingerprint: "OLD_ZZ", Country: "ZZ", Score: 0.3},
			},
		},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	response := onionooDetailsResponse{Relays: []onionooDetailsRelay{}}
	srv := httptest.NewServer(reputationJSONHandler(t, response))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	reps, err := rc.Check(context.Background(), "zz")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if len(reps) != 1 || reps[0].Fingerprint != "OLD_ZZ" {
		t.Errorf("reps = %v, want stale cache entry", reps)
	}
}

func TestReputationChecker_Check_ContextCancelled(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := rc.Check(ctx, "us")
	if err == nil {
		t.Error("Check() expected error for cancelled context")
	}
}

func TestComputeScore(t *testing.T) {
	tests := []struct {
		name      string
		flags     []string
		uptime    int
		bandwidth int64
		flagged   bool
		isNew     bool
		want      float64
	}{
		{
			name:      "base score only",
			flags:     []string{"Exit"},
			uptime:    5,
			bandwidth: 500_000,
			flagged:   false,
			isNew:     false,
			want:      0.5,
		},
		{
			name:      "stable flag",
			flags:     []string{"Exit", "Stable"},
			uptime:    5,
			bandwidth: 500_000,
			flagged:   false,
			isNew:     false,
			want:      0.6,
		},
		{
			name:      "fast flag",
			flags:     []string{"Exit", "Fast"},
			uptime:    5,
			bandwidth: 500_000,
			flagged:   false,
			isNew:     false,
			want:      0.6,
		},
		{
			name:      "high uptime",
			flags:     []string{"Exit"},
			uptime:    45,
			bandwidth: 500_000,
			flagged:   false,
			isNew:     false,
			want:      0.6,
		},
		{
			name:      "high bandwidth",
			flags:     []string{"Exit"},
			uptime:    5,
			bandwidth: 15_000_000,
			flagged:   false,
			isNew:     false,
			want:      0.6,
		},
		{
			name:      "best relay",
			flags:     []string{"Exit", "Stable", "Fast"},
			uptime:    120,
			bandwidth: 20_000_000,
			flagged:   false,
			isNew:     false,
			want:      0.9,
		},
		{
			name:      "flagged bad",
			flags:     []string{"Exit", "Stable", "Fast"},
			uptime:    120,
			bandwidth: 20_000_000,
			flagged:   true,
			isNew:     false,
			want:      0.4,
		},
		{
			name:      "new relay",
			flags:     []string{"Exit"},
			uptime:    3,
			bandwidth: 500_000,
			flagged:   false,
			isNew:     true,
			want:      0.2,
		},
		{
			name:      "flagged and new",
			flags:     []string{"Exit"},
			uptime:    3,
			bandwidth: 500_000,
			flagged:   true,
			isNew:     true,
			want:      0.0,
		},
		{
			name:      "clamped at zero",
			flags:     []string{},
			uptime:    1,
			bandwidth: 0,
			flagged:   true,
			isNew:     true,
			want:      0.0,
		},
		{
			name:      "clamped at one",
			flags:     []string{"Exit", "Stable", "Fast"},
			uptime:    120,
			bandwidth: 20_000_000,
			flagged:   false,
			isNew:     false,
			want:      0.9,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeScore(tt.flags, tt.uptime, tt.bandwidth, tt.flagged, tt.isNew)
			if diff := got - tt.want; diff < -0.0001 || diff > 0.0001 {
				t.Errorf("computeScore() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestReputationChecker_Filter(t *testing.T) {
	reps := []ExitReputation{
		{Fingerprint: "GOOD01", Score: 0.8, IsFlagged: false, IsNew: false, UptimeDays: 30, Bandwidth: 5_000_000},
		{Fingerprint: "NEW01", Score: 0.5, IsFlagged: false, IsNew: true, UptimeDays: 3, Bandwidth: 2_000_000},
		{Fingerprint: "BAD01", Score: 0.2, IsFlagged: true, IsNew: false, UptimeDays: 60, Bandwidth: 3_000_000},
		{Fingerprint: "GOOD02", Score: 0.9, IsFlagged: false, IsNew: false, UptimeDays: 100, Bandwidth: 15_000_000},
		{Fingerprint: "SLOW01", Score: 0.4, IsFlagged: false, IsNew: false, UptimeDays: 30, Bandwidth: 500_000},
		{Fingerprint: "YOUNG01", Score: 0.6, IsFlagged: false, IsNew: false, UptimeDays: 5, Bandwidth: 3_000_000},
	}

	rc := NewReputationChecker(filepath.Join(t.TempDir(), "cache.json"))

	filtered := rc.Filter(reps)

	if len(filtered) != 2 {
		t.Fatalf("Filter() returned %d entries, want 2", len(filtered))
	}

	if filtered[0].Fingerprint != "GOOD02" {
		t.Errorf("filtered[0] = %q, want GOOD02 (highest score)", filtered[0].Fingerprint)
	}
	if filtered[1].Fingerprint != "GOOD01" {
		t.Errorf("filtered[1] = %q, want GOOD01 (second highest)", filtered[1].Fingerprint)
	}
}

func TestReputationChecker_Filter_Empty(t *testing.T) {
	rc := NewReputationChecker(filepath.Join(t.TempDir(), "cache.json"))

	filtered := rc.Filter(nil)
	if len(filtered) != 0 {
		t.Errorf("Filter(nil) returned %d entries, want 0", len(filtered))
	}
}

func TestReputationChecker_Filter_AllFiltered(t *testing.T) {
	reps := []ExitReputation{
		{Fingerprint: "FLAGGED", Score: 0.1, IsFlagged: true, IsNew: false, UptimeDays: 30, Bandwidth: 5_000_000},
		{Fingerprint: "NEW", Score: 0.2, IsFlagged: false, IsNew: true, UptimeDays: 2, Bandwidth: 5_000_000},
	}

	rc := NewReputationChecker(filepath.Join(t.TempDir(), "cache.json"))

	filtered := rc.Filter(reps)
	if len(filtered) != 0 {
		t.Errorf("Filter() returned %d entries, want 0 (all filtered)", len(filtered))
	}
}

func TestDaysSinceFirstSeen(t *testing.T) {
	tests := []struct {
		name         string
		firstSeen    string
		wantZero     bool
		wantPositive bool
	}{
		{
			name:         "valid date",
			firstSeen:    "2025-01-01",
			wantPositive: true,
		},
		{
			name:      "invalid format returns zero",
			firstSeen: "not-a-date",
			wantZero:  true,
		},
		{
			name:         "datetime format",
			firstSeen:    "2025-01-01 12:00:00",
			wantPositive: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := daysSinceFirstSeen(tt.firstSeen)
			if tt.wantZero && got != 0 {
				t.Errorf("daysSinceFirstSeen(%q) = %d, want 0", tt.firstSeen, got)
			}
			if tt.wantPositive && got <= 0 {
				t.Errorf("daysSinceFirstSeen(%q) = %d, want > 0", tt.firstSeen, got)
			}
		})
	}
}

func TestReputationChecker_APIURL(t *testing.T) {
	response := onionooDetailsResponse{
		Relays: []onionooDetailsRelay{
			{
				Fingerprint:       "URLTEST",
				Country:           "nl",
				ObservedBandwidth: 1_000_000,
				Flags:             []string{"Exit"},
				FirstSeen:         "2025-06-01",
				LastSeen:          "2026-04-02",
			},
		},
	}

	var requestedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path + "?" + r.URL.RawQuery
		reputationJSONHandler(t, response).ServeHTTP(w, r)
	}))
	defer srv.Close()

	rc := NewReputationChecker(filepath.Join(t.TempDir(), "cache.json"))
	rc.SetAPIURL(srv.URL)

	_, err := rc.Check(context.Background(), "nl")
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	if requestedPath != "/?type=relay&running=true&flag=Exit&country=nl" {
		t.Errorf("requested path = %q, want query with country param", requestedPath)
	}
}

func TestReputationChecker_InvalidJSON(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer srv.Close()

	rc := NewReputationChecker(cachePath)
	rc.SetAPIURL(srv.URL)

	_, err := rc.Check(context.Background(), "us")
	if err == nil {
		t.Error("Check() expected error for invalid JSON")
	}
}

func TestReputationChecker_ReadCache_NoFile(t *testing.T) {
	rc := NewReputationChecker(filepath.Join(t.TempDir(), "nonexistent.json"))
	cache, err := rc.readCache()
	if err != nil {
		t.Errorf("readCache() error = %v, want nil for missing file", err)
	}
	if cache != nil {
		t.Error("readCache() expected nil for missing file")
	}
}

func TestReputationChecker_ReadCache_InvalidJSON(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")
	if err := os.WriteFile(cachePath, []byte("bad json"), 0600); err != nil {
		t.Fatal(err)
	}

	rc := NewReputationChecker(cachePath)
	_, err := rc.readCache()
	if err == nil {
		t.Error("readCache() expected error for invalid JSON")
	}
}

func TestReputationChecker_WriteCache_PreservesExisting(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "reputation_cache.json")

	existing := &reputationCache{
		FetchedAt: time.Now().UTC().Add(-1 * time.Hour),
		Entries: map[string][]ExitReputation{
			"US": {{Fingerprint: "US_RELAY", Country: "US"}},
		},
	}

	rc := NewReputationChecker(cachePath)

	newReps := []ExitReputation{{Fingerprint: "DE_RELAY", Country: "DE"}}
	if err := rc.writeCache("DE", newReps, existing); err != nil {
		t.Fatalf("writeCache() error = %v", err)
	}

	cache, err := rc.readCache()
	if err != nil {
		t.Fatalf("readCache() error = %v", err)
	}

	if _, ok := cache.Entries["US"]; !ok {
		t.Error("writeCache overwrote existing US entry")
	}
	if _, ok := cache.Entries["DE"]; !ok {
		t.Error("writeCache missing new DE entry")
	}
}

func TestNewReputationChecker_Defaults(t *testing.T) {
	rc := NewReputationChecker("/tmp/test_cache.json")
	if rc.minUptime != 7 {
		t.Errorf("minUptime = %d, want 7", rc.minUptime)
	}
	if rc.minBandwidth != 1<<20 {
		t.Errorf("minBandwidth = %d, want %d", rc.minBandwidth, 1<<20)
	}
	if rc.cacheTTL != defaultReputationCacheTTL {
		t.Errorf("cacheTTL = %v, want %v", rc.cacheTTL, defaultReputationCacheTTL)
	}
	if rc.client.Timeout != 30*time.Second {
		t.Errorf("client timeout = %v, want 30s", rc.client.Timeout)
	}
	if rc.apiURL != defaultOnionooDetailsURL {
		t.Errorf("apiURL = %q, want %q", rc.apiURL, defaultOnionooDetailsURL)
	}
}

func reputationJSONHandler(t *testing.T, v interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("reputationJSONHandler marshal: %v", err)
		}
		_, _ = w.Write(data)
	}
}
