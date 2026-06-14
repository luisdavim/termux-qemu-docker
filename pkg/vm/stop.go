package vm

import (
	"context"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/retry"
)

func Stop(s *config.State) error {
	if err := stop("VM", s.GetPIDFile(), s.Profile, 30*time.Second); err != nil {
		fmt.Printf("⚠️ %v\n", err)
	}
	if err := stop("Tunnel", s.GetTunnelPIDFile(), s.Profile, 5*time.Second); err != nil {
		fmt.Printf("⚠️ %v\n", err)
	}
	if err := os.Remove(s.GetDockerSocketPath()); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️ Failed to remove Docker socket: %v\n", err)
	}
	fmt.Printf("VM workspace profile '%s' completely stopped.\n", s.Profile)
	return nil
}

func stop(name, pidFile, profile string, gracePeriod time.Duration) error {
	pid, err := readPIDFile(pidFile)
	if err != nil {
		return fmt.Errorf("%s context profile namespace '%s' reports offline", name, profile)
	}

	process, err := os.FindProcess(pid)
	if err == nil {
		fmt.Printf("🛑 Terminating %s for workspace '%s' (PID %d)...\n", name, profile, pid)
		if err := retry.WithTimeout(context.Background(), gracePeriod, time.Second, 2*time.Second, func() error {
			err := process.Signal(syscall.Signal(0))
			if err == nil {
				err = process.Signal(syscall.SIGTERM)
			}
			if err == nil {
				return fmt.Errorf("process %d still exists", pid)
			}
			if err != os.ErrProcessDone && err != os.ErrNoHandle {
				return err
			}
			return nil
		}); err != nil {
			if err := process.Kill(); err != nil {
				fmt.Printf("⚠️ Failed to  stop %s process: %v\n", name, err)
				return err
			}
		}
	}

	if err := os.Remove(pidFile); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️ Failed to remove PID file %s: %v\n", pidFile, err)
	}
	return nil
}
