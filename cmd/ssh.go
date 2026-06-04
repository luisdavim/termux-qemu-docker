package cmd

import (
	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/ssh"
)

func newSSHCmd(state *config.State) *cobra.Command {
	sshCmd := &cobra.Command{
		Use:     "ssh",
		Aliases: []string{"shell", "exec"},
		Short:   "Start a SSH session on the Docker VM",
		RunE: func(cmd *cobra.Command, args []string) error {
			return ssh.Shell(state)
		},
	}

	return sshCmd
}
