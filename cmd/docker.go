package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/ssh"
)

func newDockerCmd(state *config.State) *cobra.Command {
	dockerCmd := &cobra.Command{
		Use:                "docker",
		Aliases:            []string{"docker-compose"},
		Short:              "Run docker or docker-compose commands",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmdStr := cmd.CalledAs() + " " + strings.Join(args, " ")
			if cwd, _ := os.Getwd(); cwd != "" {
				cmdStr = fmt.Sprintf(`doas sh -c "cd %s || true; %s"`, cwd, cmdStr)
			}
			return ssh.RunInPty(state, cmdStr)
		},
	}

	return dockerCmd
}
