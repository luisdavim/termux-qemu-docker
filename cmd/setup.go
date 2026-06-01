package cmd

import (
	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/luisdavim/termux-docker/pkg/setup"
)

func newSetupCmd(state *config.State) *cobra.Command {
	var arch string

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Install required dependencies and generate baseline configurations",
		Long:  `Automatically installs qemu, and openssh via pkg, verifies BIOS locations, and writes a default config.yaml file if missing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.RunSetupEnvironment(state.Prefix, state.HomeDir, arch)
		},
	}

	cmd.Flags().StringVarP(&arch, "arch", "a", "", "QEMU architecture")

	return cmd
}
