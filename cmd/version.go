package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show SPLITTER version and detected Tor features",
		Long:  `Display the SPLITTER version and auto-detected Tor features (Conflux, HTTPTunnelPort, CGO, etc.).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("SPLITTER version: %s\n", Version)
			fmt.Println("Tor feature detection: not yet implemented (Phase 3.3)")
			return nil
		},
	}
}
