package country

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

func TestMetricsFetcher_FetchCountries_Success(t *testing.T) {
	response := onionooResponse{
		Relays: []onionooRelay{
			{Country: "us", RelayFlags: []string{"Exit", "Guard", "HSDir"}},
			{Country: "de", RelayFlags: []string{"Guard", "Stable"}},
			{Country: "fr", RelayFlags: []string{"Exit", "Fast"}},
			{Country: "nl", RelayFlags: []string{"Exit", "Guard"}},
			{Country: "us", RelayFlags: []string{"Exit"}},
			{Country: "xx", RelayFlags: []string{"Running"}},
		},
	}

	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	tmpDir := t.TempDir()
	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	expected := []string{"DE", "FR", "NL", "US"}
	if len(countries) != len(expected) {
		t.Fatalf("len(countries) = %d, want %d; got %v", len(countries), len(expected), countries)
	}
	for i, c := range expected {
		if countries[i] != c {
			t.Errorf("countries[%d] = %q, want %q", i, countries[i], c)
		}
	}
}

func TestMetricsFetcher_FetchCountries_WritesCache(t *testing.T) {
	response := onionooResponse{
		Relays: []onionooRelay{
			{Country: "us", RelayFlags: []string{"Guard"}},
		},
	}

	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	tmpDir := t.TempDir()
	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	_, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, "country_cache.json"))
	if err != nil {
		t.Fatalf("cache file not created: %v", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		t.Fatalf("cache unmarshal: %v", err)
	}
	if len(entry.Countries) != 1 || entry.Countries[0] != "US" {
		t.Errorf("cached countries = %v, want [US]", entry.Countries)
	}
	if entry.FetchedAt.IsZero() {
		t.Error("cached fetched_at is zero")
	}
}

func TestMetricsFetcher_FetchCountries_UsesCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "country_cache.json")

	cached := cacheEntry{
		FetchedAt: time.Now().UTC(),
		Countries: []string{"DE", "FR", "US"},
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

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	if len(countries) != 3 {
		t.Fatalf("len(countries) = %d, want 3", len(countries))
	}
}

func TestMetricsFetcher_FetchCountries_TTLExpired(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "country_cache.json")

	cached := cacheEntry{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Countries: []string{"ZZ"},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	response := onionooResponse{
		Relays: []onionooRelay{
			{Country: "se", RelayFlags: []string{"Exit"}},
		},
	}

	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	if len(countries) != 1 || countries[0] != "SE" {
		t.Errorf("countries = %v, want [SE]", countries)
	}
}

func TestMetricsFetcher_FetchCountries_StaleCacheOnFetchError(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "country_cache.json")

	cached := cacheEntry{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Countries: []string{"DE", "US"},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v, want stale cache fallback", err)
	}

	if len(countries) != 2 {
		t.Fatalf("len(countries) = %d, want 2 (stale cache)", len(countries))
	}
}

func TestMetricsFetcher_FetchCountries_NoCacheFetchError(t *testing.T) {
	tmpDir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	_, err := mf.FetchCountries(context.Background())
	if err == nil {
		t.Error("FetchCountries() expected error when fetch fails with no cache")
	}
}

func TestMetricsFetcher_FetchCountries_EmptyAPIResponse(t *testing.T) {
	tmpDir := t.TempDir()

	response := onionooResponse{Relays: []onionooRelay{}}
	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	_, err := mf.FetchCountries(context.Background())
	if err == nil {
		t.Error("FetchCountries() expected error for empty API response with no cache")
	}
}

func TestMetricsFetcher_FetchCountries_EmptyAPIWithStaleCache(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "country_cache.json")

	cached := cacheEntry{
		FetchedAt: time.Now().UTC().Add(-48 * time.Hour),
		Countries: []string{"NL"},
	}
	data, _ := json.Marshal(cached)
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		t.Fatal(err)
	}

	response := onionooResponse{Relays: []onionooRelay{}}
	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}
	if len(countries) != 1 || countries[0] != "NL" {
		t.Errorf("countries = %v, want [NL] from stale cache", countries)
	}
}

func TestMetricsFetcher_FetchCountries_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	_, err := mf.FetchCountries(context.Background())
	if err == nil {
		t.Error("FetchCountries() expected error for invalid JSON")
	}
}

func TestMetricsFetcher_FetchCountries_ContextCancelled(t *testing.T) {
	tmpDir := t.TempDir()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := mf.FetchCountries(ctx)
	if err == nil {
		t.Error("FetchCountries() expected error for cancelled context")
	}
}

func TestMetricsFetcher_FetchCountries_DedupAndSort(t *testing.T) {
	response := onionooResponse{
		Relays: []onionooRelay{
			{Country: "zz", RelayFlags: []string{"Guard"}},
			{Country: "aa", RelayFlags: []string{"Exit"}},
			{Country: "zz", RelayFlags: []string{"Exit"}},
			{Country: "mm", RelayFlags: []string{"Guard", "Exit"}},
		},
	}

	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	tmpDir := t.TempDir()
	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	expected := []string{"AA", "MM", "ZZ"}
	if len(countries) != len(expected) {
		t.Fatalf("len(countries) = %d, want %d", len(countries), len(expected))
	}
	for i, c := range expected {
		if countries[i] != c {
			t.Errorf("countries[%d] = %q, want %q", i, countries[i], c)
		}
	}
}

func TestMetricsFetcher_FetchCountries_FilterByFlag(t *testing.T) {
	response := onionooResponse{
		Relays: []onionooRelay{
			{Country: "us", RelayFlags: []string{"Guard"}},
			{Country: "de", RelayFlags: []string{"Exit"}},
			{Country: "fr", RelayFlags: []string{"HSDir", "Stable"}},
			{Country: "nl", RelayFlags: []string{"Running"}},
			{Country: "", RelayFlags: []string{"Guard"}},
		},
	}

	srv := httptest.NewServer(jsonHandler(t, response))
	defer srv.Close()

	tmpDir := t.TempDir()
	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL(srv.URL)

	countries, err := mf.FetchCountries(context.Background())
	if err != nil {
		t.Fatalf("FetchCountries() error = %v", err)
	}

	expected := []string{"DE", "US"}
	if len(countries) != len(expected) {
		t.Fatalf("countries = %v, want %v", countries, expected)
	}
	for i, c := range expected {
		if countries[i] != c {
			t.Errorf("countries[%d] = %q, want %q", i, countries[i], c)
		}
	}
}

func TestMetricsFetcher_ReadCache_NoFile(t *testing.T) {
	mf := NewMetricsFetcher(t.TempDir())
	entry, err := mf.readCache()
	if err != nil {
		t.Errorf("readCache() error = %v, want nil for missing file", err)
	}
	if entry != nil {
		t.Error("readCache() expected nil entry for missing file")
	}
}

func TestMetricsFetcher_ReadCache_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "country_cache.json")
	if err := os.WriteFile(cachePath, []byte("bad"), 0600); err != nil {
		t.Fatal(err)
	}

	mf := NewMetricsFetcher(tmpDir)
	_, err := mf.readCache()
	if err == nil {
		t.Error("readCache() expected error for invalid JSON")
	}
}

func TestMetricsFetcher_WriteCache_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nestedDir := filepath.Join(tmpDir, "deep", "nested")
	mf := NewMetricsFetcher(nestedDir)

	if err := mf.writeCache([]string{"US"}); err != nil {
		t.Fatalf("writeCache() error = %v", err)
	}

	if _, err := os.Stat(filepath.Join(nestedDir, "country_cache.json")); err != nil {
		t.Errorf("cache file not created: %v", err)
	}
}

func TestMetricsFetcher_FetchCountries_HttpError(t *testing.T) {
	tmpDir := t.TempDir()

	mf := NewMetricsFetcher(tmpDir)
	mf.SetMetricsURL("http://127.0.0.1:1")

	_, err := mf.FetchCountries(context.Background())
	if err == nil {
		t.Error("FetchCountries() expected error for unreachable server")
	}
}

func TestNewMetricsFetcher_Defaults(t *testing.T) {
	mf := NewMetricsFetcher("/tmp/test")
	if mf.cacheTTL != defaultCacheTTL {
		t.Errorf("cacheTTL = %v, want %v", mf.cacheTTL, defaultCacheTTL)
	}
	if mf.client.Timeout != 30*time.Second {
		t.Errorf("client timeout = %v, want 30s", mf.client.Timeout)
	}
	if mf.metricsURL != defaultMetricsURL {
		t.Errorf("metricsURL = %q, want %q", mf.metricsURL, defaultMetricsURL)
	}
}

func jsonHandler(t *testing.T, v interface{}) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatalf("jsonHandler marshal: %v", err)
		}
		_, _ = w.Write(data)
	}
}
