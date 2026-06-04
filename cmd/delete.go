package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/profiles"
)

func newDeleteCmd(state *config.State) *cobra.Command {
	var force bool
	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Remove a profile and all its associated data",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !force {
				fmt.Printf("⚠️ This will permanently delete profile '%s' and its disk image.\n", state.Profile)
				fmt.Print("Are you sure you want to proceed? (y/N): ")
				reader := bufio.NewReader(os.Stdin)
				response, _ := reader.ReadString('\n')
				response = strings.ToLower(strings.TrimSpace(response))
				if response != "y" && response != "yes" {
					return fmt.Errorf("deletion cancelled")
				}
			}

			return profiles.Delete(state)
		},
	}

	deleteCmd.Flags().BoolVarP(&force, "force", "f", false, "Force deletion without confirmation")
	return deleteCmd
}
