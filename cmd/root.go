package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
)

func NewRootCmd() *cobra.Command {
	state := &config.State{}

	rootCmd := &cobra.Command{
		Use:           "termux-qemu-docker",
		Short:         "A lightweight profile-aware container VM manager for Termux",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "help" || cmd.Name() == "completion" {
				return nil
			}

			var err error
			state.HomeDir, err = os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home dir: %w", err)
			}

			state.Prefix = os.Getenv("TERMUX__PREFIX")
			// fallback for older version of termux
			if state.Prefix == "" && strings.Contains(state.HomeDir, "termux") {
				if idx := strings.Index(state.HomeDir, "home"); idx > 0 {
					state.Prefix = filepath.Join(state.HomeDir[:idx-1], "usr")
				}
			}

			if cmd.Name() == "list" || cmd.Name() == "setup" {
				return nil
			}

			state.Cfg, err = config.LoadConfig(state.Profile, state.HomeDir)
			if err != nil {
				state.Cfg = config.NewDefaultConfig(state.Profile, state.HomeDir, state.Prefix)
			} else {
				state.Cfg.SetDefaults(state.Profile, state.HomeDir, state.Prefix)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&state.Profile, "profile", "p", "default", "The profile namespace directory framework to target")

	rootCmd.AddCommand(newStartCmd(state))
	rootCmd.AddCommand(newStopCmd(state))
	rootCmd.AddCommand(newStatusCmd(state))
	rootCmd.AddCommand(newListCmd(state))
	rootCmd.AddCommand(newSetupCmd(state))
	rootCmd.AddCommand(newDeleteCmd(state))
	rootCmd.AddCommand(newTunnelCmd(state))
	rootCmd.AddCommand(newLogsCmd(state))
	rootCmd.AddCommand(newSSHCmd(state))
	rootCmd.AddCommand(newPortsCmd(state))

	return rootCmd
}
