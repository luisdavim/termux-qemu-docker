package cmd

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/spf13/cobra"
)

func newStatusCmd(state *config.State) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display the current operational state of a profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			return showStatus(state)
		},
	}
}

func showStatus(state *config.State) error {
	data, err := os.ReadFile(state.GetPIDFile())
	if err != nil {
		fmt.Printf("Profile '%s' is currently OFFLINE\n", state.Profile)
		return nil
	}

	pid, _ := strconv.Atoi(string(data))
	process, err := os.FindProcess(pid)
	if err == nil && process.Signal(syscall.Signal(0)) == nil {
		fmt.Printf("Profile '%s' is RUNNING (PID: %d)\n", state.Profile, pid)
		fmt.Printf("CPU: %d Cores | Memory: %sMB | SSH Port: %d\n", state.Cfg.VM.CPUs, state.Cfg.VM.Memory, state.Cfg.VM.SSHPort)
	} else {
		fmt.Printf("Profile '%s' is STALE (Process %d not found). Cleaning up...\n", state.Profile, pid)
		_ = os.Remove(state.GetPIDFile())
	}

	return nil
}
