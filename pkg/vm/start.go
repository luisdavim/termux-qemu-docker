package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func Start(s *config.State) error {
	if _, err := os.Stat(s.GetPIDFile()); err == nil {
		return fmt.Errorf("the profile instance '%s' appears to be already active", s.Profile)
	}

	c := s.Cfg
	if err := CheckAndDownloadImage(c); err != nil {
		return fmt.Errorf("image setup failed: %w", err)
	}

	seedISO, err := CreateSeedISO(s)
	if err != nil {
		return fmt.Errorf("cloud-init setup failed: %w", err)
	}

	if err := StartQEMU(s, seedISO); err != nil {
		return err
	}

	fmt.Printf("🌀 Spawning isolated profile namespace [%s] (%d Cores, %sMB RAM)...\n", s.Profile, c.VM.CPUs, c.VM.Memory)

	if _, err := strconv.Atoi(c.VM.Memory); err != nil {
		return fmt.Errorf("invalid memory configuration: %s. Must be numeric (MB)", c.VM.Memory)
	}

	socketDir := filepath.Dir(s.GetDockerSocketPath())
	if err := os.MkdirAll(socketDir, 0o755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	OrchestrateEnvironment(ctx, s)

	if err := startTunnel(s); err != nil {
		return fmt.Errorf("failed to start portforward tunnel: %w", err)
	}

	fmt.Println("🩺 Running Docker daemon framework availability health checks...")
	if VerifyDockerHealth(s) {
		fmt.Printf("\n✅ Profile context '%s' initialized and healthy!\n", s.Profile)
		fmt.Println("👉 Execute this declaration statement locally to connect your shell:")
		fmt.Printf(" export DOCKER_HOST=unix://%s\n\n", s.GetDockerSocketPath())
	} else {
		_ = Stop(s)
		return fmt.Errorf("health diagnostic failed. Docker daemon may still be starting or misconfigured")
	}

	return nil
}

func getQEMUCmd(s *config.State) string {
	return fmt.Sprintf("qemu-system-%s", s.Cfg.AlpineSetup.Arch)
}

func StartQEMU(s *config.State, seedISO string) error {
	args := []string{
		"-M", "virt", "-cpu", "max", "-smp", strconv.Itoa(s.Cfg.VM.CPUs), "-m", s.Cfg.VM.Memory,
		"-bios", s.Cfg.VM.BiosPath,
		"-drive", fmt.Sprintf("if=virtio,file=%s,format=qcow2", s.Cfg.VM.DiskPath),
		"-drive", fmt.Sprintf("if=virtio,file=%s,format=raw,readonly=on", seedISO),
		"-netdev", fmt.Sprintf("user,id=n1,hostfwd=tcp::%d-:22", s.Cfg.VM.SSHPort),
		"-device", "virtio-net-pci,netdev=n1", "-nographic",
	}

	for i, m := range s.Cfg.Mounts {
		if err := os.MkdirAll(m, 0o755); err != nil {
			fmt.Printf("⚠️ Warning: failed to create host mount directory %s: %v\n", m, err)
		}
		tag := fmt.Sprintf("mount%d", i)
		args = append(args,
			"-fsdev", fmt.Sprintf("local,id=%s,path=%s,security_model=none", tag, m),
			"-device", fmt.Sprintf("virtio-9p-pci,fsdev=%s,mount_tag=%s", tag, tag),
		)
	}

	qemuLogPath := s.GetLogPath()
	qemuLog, err := os.OpenFile(qemuLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open QEMU log file: %w", err)
	}

	qemuCmd := exec.Command(getQEMUCmd(s), args...)
	qemuCmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	qemuCmd.Stdout, qemuCmd.Stderr = qemuLog, qemuLog

	if err := qemuCmd.Start(); err != nil {
		return fmt.Errorf("VM error: %w", err)
	}

	// Flag to track if startup was fully successful
	startupSuccessful := false
	defer func() {
		if !startupSuccessful {
			fmt.Println("⚠️ Startup failed. Cleaning up stale VM references...")
			_ = qemuCmd.Process.Kill()
			_ = os.Remove(s.GetPIDFile())
		}
	}()

	if err := os.WriteFile(s.GetPIDFile(), []byte(strconv.Itoa(qemuCmd.Process.Pid)), 0o644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	fmt.Printf("🚀 QEMU process started (PID: %d). Orchestrating environment...\n", qemuCmd.Process.Pid)
	startupSuccessful = true

	return nil
}

func startTunnel(s *config.State) error {
	pidFile := s.GetTunnelPIDFile()
	if _, err := os.Stat(pidFile); err == nil {
		return nil
	}

	logPath := s.GetTunnelLogPath()
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open Tunnel log file: %w", err)
	}

	slef, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(slef, "tunnel")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout, cmd.Stderr = log, log

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("VM error: %w", err)
	}

	// Flag to track if startup was fully successful
	startupSuccessful := false
	defer func() {
		if !startupSuccessful {
			fmt.Println("⚠️ Startup failed. Cleaning up stale tunnel references...")
			_ = cmd.Process.Kill()
			_ = os.Remove(pidFile)
		}
	}()

	if err := os.WriteFile(pidFile, []byte(strconv.Itoa(cmd.Process.Pid)), 0o644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	startupSuccessful = true
	fmt.Printf("🚀 Tunnel process started (PID: %d).\n", cmd.Process.Pid)

	return nil
}
