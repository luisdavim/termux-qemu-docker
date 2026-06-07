package vm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
)

func Start(s *config.State) (rerr error) {
	if _, err := os.Stat(s.GetPIDFile()); err == nil {
		return fmt.Errorf("the profile instance '%s' appears to be already active", s.Profile)
	}

	c := s.Cfg
	if err := CheckAndDownloadImage(c); err != nil {
		return fmt.Errorf("image setup failed: %w", err)
	}

	seedISO, err := CreateSeedISO(s)
	if err != nil {
		return fmt.Errorf("bootstrap setup failed: %w", err)
	}

	fmt.Printf("🌀 Spawning isolated profile namespace [%s] (%d Cores, %sMB RAM)...\n", s.Profile, c.VM.CPUs, c.VM.Memory)

	if err := StartQEMU(s, seedISO); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		cancel()
		if rerr != nil {
			if err := Stop(s); err != nil {
				fmt.Printf("⚠️ Warning: tfailed to  gracefully  stop: %v\n", err)
			}
		}
	}()

	if err := OrchestrateEnvironment(ctx, s); err != nil {
		return err
	}

	if err := startTunnel(s); err != nil {
		return fmt.Errorf("failed to start portforward tunnel: %w", err)
	}

	fmt.Println("🩺 Running Docker daemon framework availability health checks...")
	if VerifyDockerHealth(s) {
		fmt.Printf("\n✅ Profile context '%s' initialized and healthy!\n", s.Profile)
		fmt.Println("👉 Execute this declaration statement locally to connect your shell:")
		fmt.Printf(" export DOCKER_HOST=unix://%s\n\n", s.GetDockerSocketPath())
	} else {
		return fmt.Errorf("health diagnostic failed. Docker daemon may still be starting or misconfigured")
	}

	return nil
}

func getQEMUCmd(s *config.State) string {
	return fmt.Sprintf("qemu-system-%s", s.Cfg.AlpineSetup.Arch)
}

func StartQEMU(s *config.State, seedISO string) error {
	machine := "virt"
	netDevice := "virtio-net-pci"
	blkDevice := "virtio-blk-pci"
	fsDevice := "virtio-9p-pci"
	rngDevice := "virtio-rng-pci"

	if s.Cfg.AlpineSetup.Arch == "x86_64" {
		machine = "q35"
	}

	args := []string{
		"-M", machine, "-cpu", "max", "-smp", strconv.Itoa(s.Cfg.VM.CPUs), "-m", s.Cfg.VM.Memory, "-nographic",
		"-bios", s.Cfg.VM.BiosPath,
		// Drop QEMU & UEFI Firmware delays down to 0ms
		"-boot", "menu=on,splash-time=0",
		"-fw_cfg", "name=opt/org.tianocore.BdsSkipSlightDelay,string=1",
		"-fw_cfg", "name=opt/org.gnu.grub.timeout,string=0",
		// Storage allocation
		"-object", "iothread,id=iothread0",
		"-device", fmt.Sprintf("%s,drive=hd0,iothread=iothread0,num-queues=%d", blkDevice, s.Cfg.VM.CPUs),
		"-drive", fmt.Sprintf("if=none,id=hd0,file=%s,format=raw,cache=unsafe,discard=on,aio=threads", s.Cfg.VM.DiskPath),
		"-drive", fmt.Sprintf("if=virtio,file=%s,format=raw,readonly=on", seedISO),
		// Network & System Hardware Layout
		"-netdev", fmt.Sprintf("user,id=n1,hostfwd=tcp::%d-:22", s.Cfg.VM.SSHPort),
		"-device", fmt.Sprintf("%s,netdev=n1", netDevice),
		"-object", "rng-random,id=rng0,filename=/dev/urandom",
		"-device", fmt.Sprintf("%s,rng=rng0", rngDevice),
	}

	if s.Cfg.VM.UseKVM {
		args = append(args, "-enable-kvm")
	} else {
		args = append(args, "-accel", "tcg,thread=multi,tb-size=256")
	}

	for i, m := range s.Cfg.Mounts {
		if err := os.MkdirAll(m, 0o755); err != nil {
			fmt.Printf("⚠️ Warning: failed to create host mount directory %s: %v\n", m, err)
		}
		tag := fmt.Sprintf("mount%d", i)
		args = append(args,
			"-fsdev", fmt.Sprintf("local,id=%s,path=%s,security_model=none", tag, m),
			"-device", fmt.Sprintf("%s,fsdev=%s,mount_tag=%s", fsDevice, tag, tag),
		)
	}

	pid, err := runInBackground(getQEMUCmd(s), s.GetPIDFile(), s.GetLogPath(), args...)
	if err != nil {
		return fmt.Errorf("failed to start VM: %w", err)
	}

	// avoid wasting resources by delaying the ssh poll
	time.Sleep(30 * time.Second)
	fmt.Printf("🚀 QEMU process started (PID: %d). Orchestrating environment...\n", pid)

	return nil
}

func startTunnel(s *config.State) error {
	slef, err := os.Executable()
	if err != nil {
		return err
	}

	pid, err := runInBackground(slef, s.GetTunnelPIDFile(), s.GetTunnelLogPath(), "tunnel", "-p", s.Profile)
	if err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	fmt.Printf("🚀 Tunnel process started (PID: %d).\n", pid)
	return nil
}

func runInBackground(name, pidFile, logFile string, args ...string) (pid int, rerr error) {
	if _, err := os.Stat(pidFile); err == nil {
		return -1, nil
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

	if cmd.ProcessState != nil {
		return -1, fmt.Errorf("process terminated with %d", cmd.ProcessState.ExitCode())
	}

	return cmd.Process.Pid, nil
}
