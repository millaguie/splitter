package tor

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type BridgeConfig struct {
	Description string   `yaml:"description"`
	Transport   string   `yaml:"transport"`
	Lines       []string `yaml:"lines"`
}

type BridgesConfig struct {
	Snowflake *BridgeConfig `yaml:"snowflake"`
	WebTunnel *BridgeConfig `yaml:"webtunnel"`
	Obfs4     *BridgeConfig `yaml:"obfs4"`
}

func LoadBridges(path string) (*BridgesConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("LoadBridges: %w", err)
	}
	var cfg BridgesConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("LoadBridges: unmarshal: %w", err)
	}
	return &cfg, nil
}

func (b *BridgesConfig) GetBridge(bridgeType string) (*BridgeConfig, error) {
	switch bridgeType {
	case "snowflake":
		return b.Snowflake, nil
	case "webtunnel":
		return b.WebTunnel, nil
	case "obfs4":
		return b.Obfs4, nil
	case "none", "":
		return nil, nil
	default:
		return nil, fmt.Errorf("GetBridge: unknown bridge type %q", bridgeType)
	}
}
