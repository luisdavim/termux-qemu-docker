package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the tool version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s %s\n", cmd.Parent().Name(), version)
		},
	}
}
