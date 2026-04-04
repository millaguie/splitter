package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/user/splitter/internal/config"
)

type CobraFlagReader struct {
	flags *pflag.FlagSet
}

func NewCobraFlagReader(flags *pflag.FlagSet) *CobraFlagReader {
	return &CobraFlagReader{flags: flags}
}

func (r *CobraFlagReader) Changed(name string) bool {
	return r.flags.Changed(name)
}

func (r *CobraFlagReader) GetInt(name string) (int, error) {
	return r.flags.GetInt(name)
}

func (r *CobraFlagReader) GetString(name string) (string, error) {
	return r.flags.GetString(name)
}

func (r *CobraFlagReader) GetBool(name string) (bool, error) {
	return r.flags.GetBool(name)
}

func Load(cmd *cobra.Command) (*config.Config, error) {
	return config.Load(config.LoadOptions{
		ConfigPath: "configs/default.yaml",
		Flags:      NewCobraFlagReader(cmd.Flags()),
	})
}
