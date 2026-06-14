package vm

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"
)

func readPIDFile(pidFile string) (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return -1, err
	}

	return strconv.Atoi(string(data))
}

func findProcess(pidFile string) (*os.Process, error) {
	pid, err := readPIDFile(pidFile)
	if err != nil {
		return nil, err
	}

	return os.FindProcess(pid)
}

func isRunning(pidFile string) (int, error) {
	var err error

	if process, _ := findProcess(pidFile); process != nil {
		if err = process.Signal(syscall.Signal(0)); err == nil {
			return process.Pid, nil
		}
	}

	return -1, err
}

func runInBackground(name, pidFile, logFile string, delayCheck time.Duration, args ...string) (pid int, rerr error) {
	if pid, _ := isRunning(pidFile); pid > 0 {
		return pid, nil
	}

	log, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return -1, fmt.Errorf("failed to open Tunnel log file: %w", err)
	}

	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout, cmd.Stderr = log, log

	if err := cmd.Start(); err != nil {
		return -1, fmt.Errorf("%s error: %w", name, err)
	}

	defer func() {
		if rerr != nil {
			fmt.Println("⚠️ Startup failed. Cleaning up stale references...")
			_ = cmd.Process.Kill()
			_ = os.Remove(pidFile)
		}
	}()

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		return -1, fmt.Errorf("failed to write PID file: %w", err)
	}

	if delayCheck > 0 {
		// the process may start and then exit with some error after some seconds
		// so we wait a bit before checking the process state
		time.Sleep(delayCheck)
	}

	if cmd.ProcessState != nil {
		return -1, fmt.Errorf("process terminated with %d", cmd.ProcessState.ExitCode())
	}

	if _, err := isRunning(pidFile); err != nil {
		return -1, fmt.Errorf("%s prematurely terminated: %w", name, err)
	}

	return cmd.Process.Pid, nil
}
