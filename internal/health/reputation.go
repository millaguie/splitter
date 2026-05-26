package health

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

const defaultReputationCacheTTL = 12 * time.Hour

const defaultOnionooDetailsURL = "https://onionoo.torproject.org/details"

type ExitReputation struct {
	Fingerprint string   `json:"fingerprint"`
	Country     string   `json:"country"`
	UptimeDays  int      `json:"uptime_days"`
	Bandwidth   int64    `json:"bandwidth"`
	Flags       []string `json:"flags"`
	IsFlagged   bool     `json:"is_flagged"`
	IsNew       bool     `json:"is_new"`
	Score       float64  `json:"score"`
}

type reputationCache struct {
	FetchedAt time.Time                   `json:"fetched_at"`
	Entries   map[string][]ExitReputation `json:"entries"`
}

type ReputationChecker struct {
	client       *http.Client
	minUptime    int
	minBandwidth int64
	cacheTTL     time.Duration
	cachePath    string
	apiURL       string
}

func NewReputationChecker(cachePath string) *ReputationChecker {
	return &ReputationChecker{
		client:       &http.Client{Timeout: 30 * time.Second},
		minUptime:    7,
		minBandwidth: 1 << 20,
		cacheTTL:     defaultReputationCacheTTL,
		cachePath:    cachePath,
		apiURL:       defaultOnionooDetailsURL,
	}
}

func (rc *ReputationChecker) SetAPIURL(url string) {
	rc.apiURL = url
}

func (rc *ReputationChecker) Check(ctx context.Context, country string) ([]ExitReputation, error) {
	upper := strings.ToUpper(country)

	cached, err := rc.readCache()
	if err == nil && cached != nil {
		if entries, ok := cached.Entries[upper]; ok && time.Since(cached.FetchedAt) < rc.cacheTTL {
			return entries, nil
		}
	}

	reputations, fetchErr := rc.fetchFromAPI(ctx, upper)
	if fetchErr != nil {
		if cached != nil {
			if entries, ok := cached.Entries[upper]; ok {
				slog.Warn("reputation fetch failed, using stale cache", "error", fetchErr, "country", upper)
				return entries, nil
			}
		}
		return nil, fmt.Errorf("Check: fetch failed and no cache for country %s: %w", upper, fetchErr)
	}

	if len(reputations) == 0 {
		if cached != nil {
			if entries, ok := cached.Entries[upper]; ok {
				slog.Warn("reputation returned empty, using cache", "country", upper)
				return entries, nil
			}
		}
		return nil, fmt.Errorf("Check: no exit relays found for country %s and no cache available", upper)
	}

	if err := rc.writeCache(upper, reputations, cached); err != nil {
		slog.Warn("failed to write reputation cache", "error", err)
	}

	return reputations, nil
}

func (rc *ReputationChecker) Filter(reputations []ExitReputation) []ExitReputation {
	var filtered []ExitReputation
	for _, r := range reputations {
		if r.IsFlagged {
			continue
		}
		if r.IsNew {
			continue
		}
		if r.UptimeDays < rc.minUptime {
			continue
		}
		if r.Bandwidth < rc.minBandwidth {
			continue
		}
		filtered = append(filtered, r)
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].Score > filtered[j].Score
	})

	return filtered
}

type onionooDetailsResponse struct {
	Relays []onionooDetailsRelay `json:"relays"`
}

type onionooDetailsRelay struct {
	Fingerprint       string   `json:"fingerprint"`
	Country           string   `json:"country"`
	ObservedBandwidth int64    `json:"observed_bandwidth"`
	Flags             []string `json:"flags"`
	FirstSeen         string   `json:"first_seen"`
	LastSeen          string   `json:"last_seen"`
}

func (rc *ReputationChecker) fetchFromAPI(ctx context.Context, country string) ([]ExitReputation, error) {
	url := fmt.Sprintf("%s?type=relay&running=true&flag=Exit&country=%s", rc.apiURL, strings.ToLower(country))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchFromAPI: creating request: %w", err)
	}

	resp, err := rc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchFromAPI: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetchFromAPI: unexpected status %d", resp.StatusCode)
	}

	var body onionooDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("fetchFromAPI: decoding response: %w", err)
	}

	reputations := make([]ExitReputation, 0, len(body.Relays))
	for _, relay := range body.Relays {
		uptimeDays := daysSinceFirstSeen(relay.FirstSeen)
		isNew := uptimeDays < rc.minUptime
		score := computeScore(relay.Flags, uptimeDays, relay.ObservedBandwidth, false, isNew)

		reputations = append(reputations, ExitReputation{
			Fingerprint: relay.Fingerprint,
			Country:     strings.ToUpper(relay.Country),
			UptimeDays:  uptimeDays,
			Bandwidth:   relay.ObservedBandwidth,
			Flags:       relay.Flags,
			IsFlagged:   false,
			IsNew:       isNew,
			Score:       score,
		})
	}

	return reputations, nil
}

func computeScore(flags []string, uptimeDays int, bandwidth int64, isFlagged bool, isNew bool) float64 {
	score := 0.5

	hasFlag := func(name string) bool {
		for _, f := range flags {
			if f == name {
				return true
			}
		}
		return false
	}

	if hasFlag("Stable") {
		score += 0.1
	}
	if hasFlag("Fast") {
		score += 0.1
	}
	if uptimeDays > 30 {
		score += 0.1
	}
	if bandwidth > 10*1<<20 {
		score += 0.1
	}
	if isFlagged {
		score -= 0.5
	}
	if isNew {
		score -= 0.3
	}

	if score < 0.0 {
		score = 0.0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func daysSinceFirstSeen(firstSeen string) int {
	t, err := time.Parse("2006-01-02", firstSeen)
	if err != nil {
		t, err = time.Parse("2006-01-02 15:04:05", firstSeen)
		if err != nil {
			return 0
		}
	}
	now := time.Now().UTC()
	days := int(now.Sub(t).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

func (rc *ReputationChecker) readCache() (*reputationCache, error) {
	data, err := os.ReadFile(rc.cachePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("readCache: %w", err)
	}

	var cache reputationCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("readCache: unmarshal: %w", err)
	}
	return &cache, nil
}

func (rc *ReputationChecker) writeCache(country string, reputations []ExitReputation, existing *reputationCache) error {
	if err := os.MkdirAll(dirOf(rc.cachePath), 0700); err != nil {
		return fmt.Errorf("writeCache: mkdir: %w", err)
	}

	cache := reputationCache{
		FetchedAt: time.Now().UTC(),
		Entries:   make(map[string][]ExitReputation),
	}

	if existing != nil {
		for k, v := range existing.Entries {
			cache.Entries[k] = v
		}
	}
	cache.Entries[country] = reputations

	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("writeCache: marshal: %w", err)
	}

	if err := os.WriteFile(rc.cachePath, data, 0600); err != nil {
		return fmt.Errorf("writeCache: write: %w", err)
	}
	return nil
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "."
	}
	return path[:idx]
}
