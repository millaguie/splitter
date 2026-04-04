package proxy

import (
	"context"
	"fmt"
)

type NativeProxy struct{}

func (p *NativeProxy) Setup(_ context.Context, instances []Instance) ([]int, error) {
	ports := make([]int, len(instances))
	for i, inst := range instances {
		if inst.HTTPPort <= 0 {
			return nil, fmt.Errorf("Setup: instance %d has no HTTPTunnelPort (HTTPPort=%d); use --proxy-mode legacy instead", inst.ID, inst.HTTPPort)
		}
		ports[i] = inst.HTTPPort
	}
	return ports, nil
}

func (p *NativeProxy) Start(_ context.Context) error {
	return nil
}

func (p *NativeProxy) Stop(_ context.Context) error {
	return nil
}

func (p *NativeProxy) Mode() Mode {
	return ModeNative
}
