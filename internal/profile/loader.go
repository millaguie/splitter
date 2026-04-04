package profile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (map[string]*Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Load: reading %s: %w", path, err)
	}

	var profiles map[string]*Profile
	if err := yaml.Unmarshal(data, &profiles); err != nil {
		return nil, fmt.Errorf("Load: parsing %s: %w", path, err)
	}

	return profiles, nil
}
