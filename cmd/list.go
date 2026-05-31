package cmd

import (
	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/luisdavim/termux-docker/pkg/profiles"
)

func newListCmd(state *config.State) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available profiles and their statuses",
		RunE: func(cmd *cobra.Command, args []string) error {
			return profiles.List(state)
		},
	}
}
