package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/user/splitter/internal/health"
)

const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiGray   = "\033[90m"
	ansiBold   = "\033[1m"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show live status of SPLITTER instances",
		Long: `Display a live dashboard showing the state of all Tor instances,
their assigned countries, circuit counts, and health status.`,
		RunE: runStatus,
	}
	cmd.Flags().String("status-url", "http://localhost:63540/status", "URL of the SPLITTER status endpoint")
	return cmd
}

func runStatus(cmd *cobra.Command, args []string) error {
	url, _ := cmd.Flags().GetString("status-url")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("runStatus: cannot connect to SPLITTER at %s (is 'splitter run' started?): %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("runStatus: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var status health.SystemStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return fmt.Errorf("runStatus: decode response: %w", err)
	}

	fmt.Print(renderStatus(&status))
	return nil
}

func renderStatus(s *health.SystemStatus) string {
	var b strings.Builder

	fmt.Fprintf(&b, "%sSPLITTER Status%s (%s)\n", ansiBold, ansiReset, s.Timestamp)
	fmt.Fprintf(&b, "═══════════════════════════════════════════\n")
	fmt.Fprintf(&b, "Tor version:  %s\n", s.TorVersion)
	fmt.Fprintf(&b, "Features:     %s\n", formatFeatures(s.Features))

	fmt.Fprintf(&b, "\nInstances (%d total, %s%d ready%s, %s%d failed%s):\n",
		s.TotalInstances,
		ansiGreen, s.ReadyCount, ansiReset,
		ansiRed, s.FailedCount, ansiReset,
	)
	fmt.Fprintf(&b, "──────────────────────────────────────────\n")

	for _, inst := range s.Instances {
		icon, color := stateIcon(inst.State)
		fmt.Fprintf(&b, "  #%d  %s  %s%s%-12s%s  socks:%d  ctrl:%d  http:%d\n",
			inst.ID,
			inst.Country,
			color, icon+" ", inst.State, ansiReset,
			inst.SocksPort,
			inst.ControlPort,
			inst.HTTPPort,
		)
	}

	if len(s.ProcessBreakdown) > 0 {
		parts := make([]string, 0, len(s.ProcessBreakdown))
		for name, count := range s.ProcessBreakdown {
			parts = append(parts, fmt.Sprintf("%s:%d", name, count))
		}
		fmt.Fprintf(&b, "\nProcesses: %d (%s)\n", s.Processes, strings.Join(parts, " "))
	} else {
		fmt.Fprintf(&b, "\nProcesses: %d\n", s.Processes)
	}

	return b.String()
}

func stateIcon(state string) (icon, color string) {
	switch state {
	case "ready":
		return "●", ansiGreen
	case "failed":
		return "○", ansiRed
	case "bootstrapping":
		return "◎", ansiYellow
	default:
		return "◦", ansiGray
	}
}

func formatFeatures(features map[string]bool) string {
	type feat struct {
		name string
		on   bool
	}
	order := []feat{
		{"conflux", features["conflux"]},
		{"http_tunnel", features["http_tunnel"]},
		{"congestion_control", features["congestion_control"]},
		{"cgo", features["cgo"]},
	}

	parts := make([]string, len(order))
	for i, f := range order {
		if f.on {
			parts[i] = fmt.Sprintf("%s%s ✓%s", ansiGreen, f.name, ansiReset)
		} else {
			parts[i] = fmt.Sprintf("%s%s ✗%s", ansiRed, f.name, ansiReset)
		}
	}
	return strings.Join(parts, " | ")
}
