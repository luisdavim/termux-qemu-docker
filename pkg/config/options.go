package config

import "github.com/spf13/cobra"

type Options struct {
	Arch   string
	CPUs   int
	Memory string
	Disk   int
	UseKVM bool
}

func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().IntVarP(&o.CPUs, "cpus", "c", 0, "Number of CPU cores")
	cmd.Flags().StringVarP(&o.Memory, "memory", "m", "", "RAM allocation in MB")
	cmd.Flags().IntVarP(&o.Disk, "disk", "d", 0, "Disk size in GB")
	cmd.Flags().StringVarP(&o.Arch, "arch", "a", "", "QEMU architecture")
	cmd.Flags().BoolVar(&o.UseKVM, "use-kvm", false, "Enable KVM support, not all devices support it")
}
