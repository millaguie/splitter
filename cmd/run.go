package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/splitter/internal/circuit"
	"github.com/user/splitter/internal/config"
	"github.com/user/splitter/internal/country"
	"github.com/user/splitter/internal/haproxy"
	"github.com/user/splitter/internal/health"
	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

type torRotator struct {
	tm *tor.TorManager
}

func (r *torRotator) GetInstances() []country.InstanceInfo {
	infos := r.tm.GetInstanceInfos()
	out := make([]country.InstanceInfo, len(infos))
	for i, info := range infos {
		out[i] = country.InstanceInfo{ID: info.ID, Country: info.Country}
	}
	return out
}

func (r *torRotator) RotateInstance(ctx context.Context, id int, newCountry string) error {
	return r.tm.RotateInstance(ctx, id, newCountry)
}

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start SPLITTER with Tor instances and HAProxy load balancing",
		Long: `Start SPLITTER by spawning the configured number of Tor instances,
generating HAProxy configuration, and launching the load balancer.
Country rotation and circuit renewal daemons start in the background.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("country-interval") {
				v, _ := cmd.Flags().GetDuration("country-interval")
				appCfg.Country.Rotation.Interval = int(v.Seconds())
			}
			if cmd.Flags().Changed("load-balance") {
				v, _ := cmd.Flags().GetString("load-balance")
				appCfg.Proxy.LoadBalanceAlgorithm = v
			}

			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			tmpDir := appCfg.Paths.TempFiles

			if err := os.MkdirAll(tmpDir, 0700); err != nil {
				return fmt.Errorf("mkdir %s: %w", tmpDir, err)
			}

			cleanupMgr := process.NewManager(tmpDir)
			_ = cleanupMgr.StopAll(ctx)
			_ = cleanupMgr.Cleanup()

			if err := os.MkdirAll(tmpDir, 0700); err != nil {
				return fmt.Errorf("mkdir %s: %w", tmpDir, err)
			}

			procMgr := process.NewManager(tmpDir)

			checkResult, err := health.CheckDependencies(ctx, appCfg)
			if err != nil {
				return err
			}

			countries, err := country.SelectRandom(appCfg.Country.Accepted, appCfg.Country.Blacklisted, appCfg.Instances.Countries)
			if err != nil {
				return fmt.Errorf("country selection: %w", err)
			}

			torMgr := tor.NewManager(appCfg, procMgr)
			if err := torMgr.DetectAndCreate(ctx, countries); err != nil {
				return fmt.Errorf("tor init: %w", err)
			}

			if err := torMgr.StartAllWithRestart(ctx); err != nil {
				return fmt.Errorf("tor start: %w", err)
			}

			renewer := circuit.NewRenewer()
			for _, inst := range torMgr.GetInstances() {
				cookiePath := filepath.Join(tmpDir, fmt.Sprintf("tor_data_%d", inst.ID), "control_auth_cookie")
				renewer.AddInstance(inst.ID, inst.ControlPort, cookiePath)
			}
			if err := renewer.Start(ctx); err != nil {
				return fmt.Errorf("circuit renewal: %w", err)
			}

			haproxyMgr := haproxy.NewManager(appCfg, procMgr)
			if err := haproxyMgr.GenerateConfig(torMgr.GetInstances()); err != nil {
				return fmt.Errorf("haproxy config: %w", err)
			}
			if err := haproxyMgr.Start(ctx); err != nil {
				return fmt.Errorf("haproxy start: %w", err)
			}

			rotator := &torRotator{tm: torMgr}
			countryDaemon := country.NewDaemon(appCfg, rotator)
			if err := countryDaemon.Start(ctx); err != nil {
				return fmt.Errorf("country daemon: %w", err)
			}

			printStartupInfo(torMgr, haproxyMgr, checkResult)

			statusPort, _ := cmd.Flags().GetInt("status-port")
			if statusPort > 0 {
				startStatusServer(ctx, statusPort, torMgr, procMgr)
			}

			sighupCh := make(chan os.Signal, 1)
			signal.Notify(sighupCh, syscall.SIGHUP)
			defer signal.Stop(sighupCh)

			for {
				select {
				case <-ctx.Done():
					fmt.Println("\nShutting down...")

					shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer shutdownCancel()

					_ = countryDaemon.Stop()
					_ = renewer.Stop()
					_ = haproxyMgr.Stop(shutdownCtx)
					_ = torMgr.StopAll(shutdownCtx)
					_ = procMgr.StopAll(shutdownCtx)
					_ = procMgr.Cleanup()

					fmt.Println("SPLITTER stopped.")
					return nil
				case sig := <-sighupCh:
					if sig == syscall.SIGHUP {
						handleReload(ctx, appCfg, torMgr, haproxyMgr, countryDaemon)
					}
				}
			}
		},
	}

	cmd.Flags().Duration("country-interval", 120*time.Second, "Interval between country rotations")
	cmd.Flags().String("load-balance", "roundrobin", "Load balancing algorithm (roundrobin|leastconn)")
	cmd.Flags().Int("status-port", health.DefaultStatusPort, "Port for status HTTP endpoint (0 to disable)")

	return cmd
}

func printStartupInfo(torMgr *tor.TorManager, haproxyMgr *haproxy.HAProxyManager, checkResult *health.CheckResult) {
	instances := torMgr.GetInstances()
	fmt.Println("=== SPLITTER ===")
	fmt.Printf("Tor instances: %d\n", len(instances))
	fmt.Printf("SOCKS proxy:   %s:%d\n", appCfg.Proxy.Master.Listen, appCfg.Proxy.Master.SocksPort)
	fmt.Printf("HTTP proxy:    %s:%d\n", appCfg.Proxy.Master.Listen, appCfg.Proxy.Master.HTTPPort)
	fmt.Printf("HAProxy stats: %s:%d%s (password: %s)\n",
		appCfg.Proxy.Stats.Listen, appCfg.Proxy.Stats.Port, appCfg.Proxy.Stats.URI,
		haproxyMgr.StatsPassword())
	fmt.Printf("Relay enforce: %s\n", appCfg.Relay.Enforce)
	fmt.Printf("Proxy mode:    %s\n", appCfg.ProxyMode)
	if v := torMgr.GetVersion(); v != nil {
		fmt.Printf("Tor version:   %s\n", v.String())
	}
	if checkResult != nil {
		f := checkResult.Features
		fmt.Printf("Features:      conflux=%v congestion=%v http-tunnel=%v cgo=%v\n",
			f.Conflux, f.CongestionControl, f.HTTPTunnel, f.CGO)
	}
	fmt.Println("================")
}

func startStatusServer(ctx context.Context, port int, torMgr *tor.TorManager, procMgr *process.Manager) {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", health.StatusHandler(torMgr, procMgr))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()

	go func() {
		slog.Info("starting status server", "port", port)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			slog.Error("status server error", "error", err)
		}
	}()
}

func handleReload(ctx context.Context, cfg *config.Config, torMgr *tor.TorManager, haproxyMgr *haproxy.HAProxyManager, countryDaemon *country.Daemon) {
	fmt.Println("Received SIGHUP, reloading configuration...")

	newCfg, err := config.Load(config.LoadOptions{
		ConfigPath: "configs/default.yaml",
		EnvPrefix:  "SPLITTER_",
	})
	if err != nil {
		fmt.Printf("Reload failed: %v\n", err)
		slog.Error("config reload failed", "error", err)
		return
	}

	changes := diffConfig(cfg, newCfg)

	if changes.CountryListChanged || changes.RotationChanged {
		countryDaemon.UpdateConfig(newCfg)
		cfg.Country = newCfg.Country
	}

	if changes.HAProxyChanged {
		cfg.Proxy = newCfg.Proxy
		cfg.HealthCheck = newCfg.HealthCheck
		cfg.Instances.Retries = newCfg.Instances.Retries
		cfg.ProxyMode = newCfg.ProxyMode
		cfg.Privoxy = newCfg.Privoxy

		if err := haproxyMgr.GenerateConfig(torMgr.GetInstances()); err != nil {
			fmt.Printf("Reload failed: HAProxy config generation: %v\n", err)
			slog.Error("haproxy config generation failed on reload", "error", err)
			return
		}
		if err := haproxyMgr.Reload(ctx); err != nil {
			fmt.Printf("Reload failed: HAProxy reload: %v\n", err)
			slog.Error("haproxy reload failed", "error", err)
			return
		}
	}

	if changes.TorChanged {
		fmt.Println("Warning: Tor configuration changed but instances require restart to apply.")
		slog.Warn("tor config changed, instances need restart")
	}

	fmt.Printf("Reloaded: %s\n", changes.Summary())
	slog.Info("configuration reloaded", "changes", changes.Summary())
}
