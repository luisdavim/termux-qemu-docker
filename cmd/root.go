package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func NewRootCmd(homeDir string) *cobra.Command {
	state := &config.State{HomeDir: homeDir}

	rootCmd := &cobra.Command{
		Use:           "termux-docker",
		Short:         "A lightweight profile-aware container VM manager for Termux",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "list" || cmd.Name() == "help" || cmd.Name() == "completion" || cmd.Name() == "setup" {
				return nil
			}

			state.Prefix = os.Getenv("TERMUX__PREFIX")

			var err error
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

	return rootCmd
}
