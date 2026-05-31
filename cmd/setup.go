package cmd

import (
	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/luisdavim/termux-docker/pkg/setup"
	"github.com/spf13/cobra"
)

func newSetupCmd(state *config.State) *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Install required dependencies and generate baseline configurations",
		Long:  `Automatically installs qemu-utils, qemu-system-aarch64-headless, and openssh via pkg, verifies BIOS locations, and writes a default config.yaml file if missing.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return setup.RunSetupEnvironment(state.HomeDir)
		},
	}
}
