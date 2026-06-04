package cmd

import (
	"time"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/ssh"
)

func newTunnelCmd(state *config.State) *cobra.Command {
	var interval time.Duration

	tunnelCmd := &cobra.Command{
		Use:   "tunnel",
		Short: "Start automatic port forwarding tunnel",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ssh.StartConnForwarder(state, interval)
		},
	}

	tunnelCmd.Flags().DurationVarP(&interval, "interval", "i", 2*time.Second, "Port check loop interval")

	return tunnelCmd
}
