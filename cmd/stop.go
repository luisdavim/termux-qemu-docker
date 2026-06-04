package cmd

import (
	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/vm"
)

func newStopCmd(state *config.State) *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Gracefully shut down the profiled engine environment",
		RunE: func(cmd *cobra.Command, args []string) error {
			return vm.Stop(state)
		},
	}
}
