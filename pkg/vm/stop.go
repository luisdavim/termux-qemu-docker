package vm

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
)

func Stop(s *config.State) error {
	if err := stop("VM", s.GetPIDFile(), s.Profile); err != nil {
		fmt.Printf("⚠️ %v\n", err)
	}
	if err := os.Remove(s.GetDockerSocketPath()); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️ Failed to remove Docker socket: %v\n", err)
	}
	if err := stop("Tunnel", s.GetTunnelPIDFile(), s.Profile); err != nil {
		fmt.Printf("⚠️ %v\n", err)
	}
	fmt.Printf("VM workspace profile '%s' completely stopped.\n", s.Profile)
	return nil
}

func stop(name, pidFile, profile string) error {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return fmt.Errorf("%s context profile namespace '%s' reports offline", name, profile)
	}

	pid, _ := strconv.Atoi(string(data))
	process, err := os.FindProcess(pid)
	if err == nil {
		fmt.Printf("🛑 Terminating %s for workspace '%s' (PID %d)...\n", name, profile, pid)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("⚠️ Failed to send SIGTERM to %s (PID %d): %v\n", name, pid, err)
		}

		for range 10 {
			time.Sleep(1 * time.Second)
			if err := process.Signal(syscall.Signal(0)); err != nil {
				break
			}
		}
		if err := process.Kill(); err != nil && err != os.ErrProcessDone {
			// Silently ignore
			_ = err
		}
	}

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️ Failed to remove PID file %s: %v\n", pidFile, err)
	}
	return nil
}
