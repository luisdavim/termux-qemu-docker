package cmd

import (
	"strings"

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
			if cmd.CalledAs() == "exec" {
				return ssh.RunInPty(state, strings.Join(args, " "))
			}
			return ssh.Shell(state)
		},
	}

	return sshCmd
}
