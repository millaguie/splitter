package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/user/splitter/internal/cli"
	"github.com/user/splitter/internal/config"
)

var appCfg *config.Config

func Execute() error {
	os.Args = cli.PreprocessArgs(os.Args)

	root := &cobra.Command{
		Use:   "splitter",
		Short: "SPLITTER manages multiple Tor instances with HAProxy load balancing",
		Long: `SPLITTER creates and manages multiple TOR network instances
load-balanced via HAProxy, with geo-based anti-correlation rules
for TOR entry/exit node selection.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			appCfg, err = cli.Load(cmd)
			if err != nil {
				return fmt.Errorf("config: %w", err)
			}
			if err := cli.SetupLogger(appCfg); err != nil {
				return fmt.Errorf("logger: %w", err)
			}
			return nil
		},
	}

	cli.BindFlags(root)

	root.AddCommand(
		newRunCmd(),
		newStatusCmd(),
		newTestCmd(),
		newVersionCmd(),
	)

	return root.Execute()
}
