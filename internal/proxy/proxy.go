package proxy

import (
	"context"
	"fmt"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/process"
)

type Mode string

const (
	ModeNative Mode = "native"
	ModeLegacy Mode = "legacy"
)

type Proxy interface {
	Setup(ctx context.Context, instances []Instance) ([]int, error)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Mode() Mode
}

type Instance struct {
	ID        int
	SocksPort int
	HTTPPort  int
}

func NewProxy(mode Mode, cfg *config.Config, procMgr *process.Manager) Proxy {
	switch mode {
	case ModeLegacy:
		return &LegacyProxy{
			cfg:     cfg,
			procMgr: procMgr,
		}
	default:
		return &NativeProxy{}
	}
}

func ParseMode(s string) (Mode, error) {
	switch s {
	case "native":
		return ModeNative, nil
	case "legacy":
		return ModeLegacy, nil
	default:
		return "", fmt.Errorf("ParseMode: unknown proxy mode %q", s)
	}
}
