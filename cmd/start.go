package cmd

import (
	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/luisdavim/termux-docker/pkg/vm"
	"github.com/spf13/cobra"
)

func newStartCmd(state *config.State) *cobra.Command {
	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a container engine VM workspace instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirty := false
			if state.CPUs > 0 {
				state.Cfg.VM.CPUs = state.CPUs
				dirty = true
			}
			if state.Memory != "" {
				state.Cfg.VM.Memory = state.Memory
				dirty = true
			}
			if state.Disk > 0 {
				state.Cfg.VM.DiskSizeGB = state.Disk
				dirty = true
			}

			if dirty {
				_ = config.SaveConfig(state.Profile, state.HomeDir, state.Cfg)
			}

			return vm.Start(state)
		},
	}

	startCmd.Flags().IntVarP(&state.CPUs, "cpus", "c", 0, "Number of CPU cores")
	startCmd.Flags().StringVarP(&state.Memory, "memory", "m", "", "RAM allocation in MB")
	startCmd.Flags().IntVarP(&state.Disk, "disk", "d", 0, "Disk size in GB")

	return startCmd
}
