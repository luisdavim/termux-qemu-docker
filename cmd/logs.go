package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
)

func newLogsCmd(state *config.State) *cobra.Command {
	var tunnel bool
	var follow bool

	logsCmd := &cobra.Command{
		Use:   "logs",
		Short: "View logs from the VM or Tunnel process",
		RunE: func(cmd *cobra.Command, args []string) error {
			logPath := state.GetLogPath()
			if tunnel {
				logPath = state.GetTunnelLogPath()
			}

			file, err := os.Open(logPath)
			if err != nil {
				return fmt.Errorf("could not open log file: %w", err)
			}
			defer func() { _ = file.Close() }()

			if !follow {
				_, err = io.Copy(os.Stdout, file)
				return err
			}

			// Simple follow implementation
			_, _ = io.Copy(os.Stdout, file)
			for {
				_, err := io.Copy(os.Stdout, file)
				if err != nil {
					return err
				}
			}
		},
	}

	logsCmd.Flags().BoolVarP(&tunnel, "tunnel", "t", false, "Show tunnel logs instead of VM logs")
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (experimental)")

	return logsCmd
}
