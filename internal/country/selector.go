package country

import (
	"fmt"
	"math/rand/v2"
)

type shuffleFunc func(n int, swap func(i, j int))

func SelectRandom(accepted, blacklisted []string, count int) ([]string, error) {
	return selectRandom(accepted, blacklisted, count, rand.Shuffle)
}

func selectRandom(accepted, blacklisted []string, count int, shuffle shuffleFunc) ([]string, error) {
	filtered := filterBlacklisted(accepted, blacklisted)
	if count > len(filtered) {
		return nil, fmt.Errorf("SelectRandom: requested %d countries, only %d available after filtering", count, len(filtered))
	}
	if count <= 0 {
		return nil, nil
	}

	result := make([]string, len(filtered))
	copy(result, filtered)
	shuffle(len(result), func(i, j int) {
		result[i], result[j] = result[j], result[i]
	})
	return result[:count], nil
}

func filterBlacklisted(accepted, blacklisted []string) []string {
	if len(blacklisted) == 0 {
		return accepted
	}

	blacklist := make(map[string]bool, len(blacklisted))
	for _, c := range blacklisted {
		blacklist[c] = true
	}

	filtered := make([]string, 0, len(accepted))
	for _, c := range accepted {
		if !blacklist[c] {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
