package cmd

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/ssh"
)

func newDockerCmd(state *config.State) *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:                "docker",
		Aliases:            []string{"docker-compose"},
		Short:              "Start a SSH session on the Docker VM",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdStr := cmd.CalledAs() + " " + strings.Join(args, " ")
			return ssh.RunInPty(state, cmdStr)
		},
	}

	return dockerCmd
}
