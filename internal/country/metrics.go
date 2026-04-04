package country

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

const defaultCacheTTL = 12 * time.Hour

const defaultMetricsURL = "https://onionoo.torproject.org/details?type=relay&running=true&fields=country,relay_flags"

type cacheEntry struct {
	FetchedAt time.Time `json:"fetched_at"`
	Countries []string  `json:"countries"`
}

type MetricsFetcher struct {
	client     *http.Client
	cacheTTL   time.Duration
	cacheDir   string
	metricsURL string
}

func NewMetricsFetcher(cacheDir string) *MetricsFetcher {
	return &MetricsFetcher{
		client:     &http.Client{Timeout: 30 * time.Second},
		cacheTTL:   defaultCacheTTL,
		cacheDir:   cacheDir,
		metricsURL: defaultMetricsURL,
	}
}

func (mf *MetricsFetcher) SetMetricsURL(url string) {
	mf.metricsURL = url
}

func (mf *MetricsFetcher) FetchCountries(ctx context.Context) ([]string, error) {
	cached, err := mf.readCache()
	if err == nil && cached != nil && time.Since(cached.FetchedAt) < mf.cacheTTL {
		return cached.Countries, nil
	}

	countries, fetchErr := mf.fetchFromAPI(ctx)
	if fetchErr != nil {
		if cached != nil {
			slog.Warn("metrics fetch failed, using stale cache", "error", fetchErr, "cache_age", time.Since(cached.FetchedAt))
			return cached.Countries, nil
		}
		return nil, fmt.Errorf("FetchCountries: fetch failed and no cache: %w", fetchErr)
	}

	if len(countries) == 0 {
		if cached != nil {
			slog.Warn("metrics returned empty country list, using cache")
			return cached.Countries, nil
		}
		return nil, fmt.Errorf("FetchCountries: API returned no countries and no cache available")
	}

	if err := mf.writeCache(countries); err != nil {
		slog.Warn("failed to write country cache", "error", err)
	}

	return countries, nil
}

type onionooResponse struct {
	Relays []onionooRelay `json:"relays"`
}

type onionooRelay struct {
	Country    string   `json:"country"`
	RelayFlags []string `json:"relay_flags"`
}

func (mf *MetricsFetcher) fetchFromAPI(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, mf.metricsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchFromAPI: creating request: %w", err)
	}

	resp, err := mf.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchFromAPI: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchFromAPI: unexpected status %d", resp.StatusCode)
	}

	var body onionooResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fetchFromAPI: decoding response: %w", err)
	}

	seen := make(map[string]bool)
	for _, relay := range body.Relays {
		if relay.Country == "" {
			continue
		}
		hasRelay := false
		for _, flag := range relay.RelayFlags {
			if flag == "Guard" || flag == "Exit" {
				hasRelay = true
				break
			}
		}
		if !hasRelay {
			continue
		}
		code := strings.ToUpper(relay.Country)
		seen[code] = true
	}

	countries := make([]string, 0, len(seen))
	for c := range seen {
		countries = append(countries, c)
	}
	sort.Strings(countries)
	return countries, nil
}

func (mf *MetricsFetcher) cachePath() string {
	return mf.cacheDir + "/country_cache.json"
}

func (mf *MetricsFetcher) readCache() (*cacheEntry, error) {
	data, err := os.ReadFile(mf.cachePath())
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readCache: %w", err)
	}

	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("readCache: unmarshal: %w", err)
	}
	return &entry, nil
}

func (mf *MetricsFetcher) writeCache(countries []string) error {
	if err := os.MkdirAll(mf.cacheDir, 0700); err != nil {
		return fmt.Errorf("writeCache: mkdir: %w", err)
	}

	entry := cacheEntry{
		FetchedAt: time.Now().UTC(),
		Countries: countries,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("writeCache: marshal: %w", err)
	}

	if err := os.WriteFile(mf.cachePath(), data, 0600); err != nil {
		return fmt.Errorf("writeCache: write: %w", err)
	}
	return nil
}
