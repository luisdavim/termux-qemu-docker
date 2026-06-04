package cmd

import (
	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/vm"
)

func newStartCmd(state *config.State) *cobra.Command {
	var opts config.Options

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Launch a container engine VM workspace instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			dirty := false
			if opts.CPUs > 0 {
				state.Cfg.VM.CPUs = opts.CPUs
				dirty = true
			}
			if opts.Memory != "" {
				state.Cfg.VM.Memory = opts.Memory
				dirty = true
			}
			if opts.Disk > 0 {
				state.Cfg.VM.DiskSizeGB = opts.Disk
				dirty = true
			}

			if opts.Arch != "" {
				state.Cfg.AlpineSetup.Arch = opts.Arch
				dirty = true
			}

			if dirty {
				_ = config.SaveConfig(state.Profile, state.HomeDir, state.Cfg)
			}

			return vm.Start(state)
		},
	}

	opts.AddFlags(startCmd)

	return startCmd
}
