package haproxy

import (
	"fmt"
	"math/rand"

	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/tor"
)

type ConfigData struct {
	Listen           string
	SOCKSPort        int
	HTTPPort         int
	StatsListen      string
	StatsPort        int
	StatsURI         string
	StatsPassword    string
	ClientTimeout    int
	ServerTimeout    int
	Retries          int
	BalanceAlgorithm string
	CheckInterval    int
	MaxFail          int
	MinSuccess       int
	HTTPBackends     []Backend
	SOCKSBackends    []Backend
}

type Backend struct {
	Name          string
	Address       string
	Port          int
	CheckInterval int
	MaxFail       int
	MinSuccess    int
}

func BuildConfigData(cfg *config.Config, instances []*tor.Instance, statsPassword string) *ConfigData {
	httpBackends := make([]Backend, 0, len(instances))
	socksBackends := make([]Backend, 0, len(instances))

	for _, inst := range instances {
		socksBackends = append(socksBackends, Backend{
			Name:          fmt.Sprintf("tor_socks_%d", inst.ID),
			Address:       "127.0.0.1",
			Port:          inst.SocksPort,
			CheckInterval: cfg.HealthCheck.Interval,
			MaxFail:       cfg.HealthCheck.MaxFail,
			MinSuccess:    cfg.HealthCheck.MinimumSuccess,
		})

		if cfg.ProxyMode == "native" {
			if inst.HTTPPort > 0 {
				httpBackends = append(httpBackends, Backend{
					Name:          fmt.Sprintf("tor_http_%d", inst.ID),
					Address:       "127.0.0.1",
					Port:          inst.HTTPPort,
					CheckInterval: cfg.HealthCheck.Interval,
					MaxFail:       cfg.HealthCheck.MaxFail,
					MinSuccess:    cfg.HealthCheck.MinimumSuccess,
				})
			}
		} else {
			privoxyPort := cfg.Privoxy.StartPort + inst.ID
			httpBackends = append(httpBackends, Backend{
				Name:          fmt.Sprintf("privoxy_%d", inst.ID),
				Address:       "127.0.0.1",
				Port:          privoxyPort,
				CheckInterval: cfg.HealthCheck.Interval,
				MaxFail:       cfg.HealthCheck.MaxFail,
				MinSuccess:    cfg.HealthCheck.MinimumSuccess,
			})
		}
	}

	rand.Shuffle(len(httpBackends), func(i, j int) {
		httpBackends[i], httpBackends[j] = httpBackends[j], httpBackends[i]
	})

	rand.Shuffle(len(socksBackends), func(i, j int) {
		socksBackends[i], socksBackends[j] = socksBackends[j], socksBackends[i]
	})

	return &ConfigData{
		Listen:           cfg.Proxy.Master.Listen,
		SOCKSPort:        cfg.Proxy.Master.SocksPort,
		HTTPPort:         cfg.Proxy.Master.HTTPPort,
		StatsListen:      cfg.Proxy.Stats.Listen,
		StatsPort:        cfg.Proxy.Stats.Port,
		StatsURI:         cfg.Proxy.Stats.URI,
		StatsPassword:    statsPassword,
		ClientTimeout:    cfg.Proxy.Master.ClientTimeout,
		ServerTimeout:    cfg.Proxy.Master.ServerTimeout,
		Retries:          cfg.Instances.Retries,
		BalanceAlgorithm: cfg.Proxy.LoadBalanceAlgorithm,
		CheckInterval:    cfg.HealthCheck.Interval,
		MaxFail:          cfg.HealthCheck.MaxFail,
		MinSuccess:       cfg.HealthCheck.MinimumSuccess,
		HTTPBackends:     httpBackends,
		SOCKSBackends:    socksBackends,
	}
}
