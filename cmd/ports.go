package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func newPortsCmd(state *config.State) *cobra.Command {
	portsCmd := &cobra.Command{
		Use:   "ports",
		Short: "List all active port mappings from the Docker VM",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := state.GetPortMapFile()
			portState, err := config.LoadPortMappings(path)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Println("No active port mappings found. Is the tunnel running?")
					return nil
				}
				return fmt.Errorf("failed to load port mappings: %w", err)
			}

			if len(portState.Mappings) == 0 {
				fmt.Println("No active port mappings found.")
				return nil
			}

			fmt.Printf("ACTIVE PORT MAPPINGS [Profile: %s]\n\n", state.Profile)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
			_, _ = fmt.Fprintln(w, "PROTO\tLOCAL ADDRESS\tVM ADDRESS\tSTATUS")

			for _, m := range portState.Mappings {
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", m.Proto, m.LocalAddress, m.VMAddress, m.Status)
			}
			_ = w.Flush()

			return nil
		},
	}

	return portsCmd
}
