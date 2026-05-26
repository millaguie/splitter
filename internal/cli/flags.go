package cli

import "github.com/spf13/cobra"

func BindFlags(cmd *cobra.Command) {
	p := cmd.PersistentFlags()

	p.IntP("instances", "i", 2, "Number of Tor instances per country")
	p.IntP("countries", "c", 6, "Number of countries to select")
	p.StringP("relay-enforce", "r", "entry", "Relay enforcement mode (entry|exit|speed) [legacy: -re]")
	p.String("profile", "", "Configuration profile (stealth|balanced|streaming|pentest)")
	p.String("proxy-mode", "native", "Proxy mode (native|legacy)")
	p.String("bridge-type", "none", "Bridge type (snowflake|webtunnel|obfs4|none)")
	p.Bool("verbose", false, "Enable verbose output")
	p.Bool("log", false, "Enable logging (off by default)")
	p.String("log-level", "info", "Log level (debug|info|warn|error)")
	p.Bool("auto-countries", false, "Auto-fetch country list from Tor Metrics API")
	p.Bool("stream-isolation", false, "Enable stream isolation via SOCKS5 auth (IsolateSOCKSAuth)")
	p.Bool("ipv6", false, "Enable IPv6 dual-stack relay selection (ClientUseIPv6)")
	p.Bool("exit-reputation", false, "Check exit node reputation before use ( Onionoo API)")
}

func PreprocessArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		if args[i] == "-re" {
			out = append(out, "--relay-enforce")
			continue
		}
		out = append(out, args[i])
	}
	return out
}
