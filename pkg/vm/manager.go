package vm

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/retry"
	"github.com/luisdavim/termux-qemu-docker/pkg/ssh"
	"github.com/luisdavim/termux-qemu-docker/pkg/utils"
)

func CheckAndDownloadImage(c *config.Config) error {
	if _, err := os.Stat(c.VM.DiskPath); err == nil {
		return nil
	}

	if c.AlpineSetup.Version == "latest" {
		fmt.Printf("🔍 Resolving latest Alpine %s version...\n", c.AlpineSetup.Arch)
		ver, err := utils.GetLatestAlpineVersion(c.AlpineSetup.Mirror, c.AlpineSetup.Arch)
		if err != nil {
			return fmt.Errorf("failed to resolve latest version: %w", err)
		}
		c.AlpineSetup.Version = ver
	}

	fmt.Printf("📂 Profile Disk allocation missing. Downloading Alpine %s %s cloud image...\n", c.AlpineSetup.Version, c.AlpineSetup.Arch)
	versionParts := strings.Split(c.AlpineSetup.Version, ".")
	majorMinor := fmt.Sprintf("v%s.%s", versionParts[0], versionParts[1])
	archSuffix := "uefi"
	if c.AlpineSetup.Arch == "x86_64" {
		archSuffix = "bios"
	}
	downloadURL := fmt.Sprintf("%s/%s/releases/cloud/nocloud_alpine-%s-%s-%s-%s-r0.qcow2",
		c.AlpineSetup.Mirror, majorMinor, c.AlpineSetup.Version, c.AlpineSetup.Arch, archSuffix, c.AlpineSetup.Bootstrap)

	tempPath := c.VM.DiskPath + ".tmp"
	if err := utils.DownloadFile(downloadURL, tempPath); err != nil {
		if remErr := os.Remove(tempPath); remErr != nil && !os.IsNotExist(remErr) {
			fmt.Printf("⚠️ Warning: failed to remove temporary download file: %v\n", remErr)
		}
		return fmt.Errorf("download failed: %v", err)
	}

	_ = os.MkdirAll(filepath.Dir(c.VM.DiskPath), 0o755)
	if err := os.Rename(tempPath, c.VM.DiskPath); err != nil {
		return err
	}

	fmt.Printf("📦 Resizing disk to %dGB...\n", c.VM.DiskSizeGB)
	cmd := exec.Command("qemu-img", "resize", c.VM.DiskPath, fmt.Sprintf("%dG", c.VM.DiskSizeGB))
	return cmd.Run()
}

func CreateSeedISO(s *config.State) (string, error) {
	tempDir, err := os.MkdirTemp("", "termux-qemu-docker-seed-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	keyPath := ssh.GetKeyPath(s.HomeDir)
	pubKeyPath := ssh.GetPublicKeyPath(s.HomeDir)
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		fmt.Println("🔑 Generating new SSH key pair for VM access...")
		if err := ssh.MakeKeyPair(pubKeyPath, keyPath); err != nil {
			return "", fmt.Errorf("failed to generate SSH key pair: %w", err)
		}
	}

	pubKeyBytes, err := os.ReadFile(pubKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read public key: %w", err)
	}

	hashedPassword, err := utils.EncryptPassword(s.Cfg.VM.SSHPassword)
	if err != nil {
		return "", err
	}

	var userData bytes.Buffer
	data := struct {
		ProfileName string
		SSHUser     string
		SSHPassword string
		PublicKey   string
	}{
		ProfileName: s.Profile,
		SSHUser:     s.Cfg.VM.SSHUser,
		SSHPassword: hashedPassword,
		PublicKey:   strings.TrimSpace(string(pubKeyBytes)),
	}

	tmpl, err := template.New("user-data").Parse(userDataTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse user-data template: %w", err)
	}
	if err := tmpl.Execute(&userData, data); err != nil {
		return "", fmt.Errorf("failed to execute user-data template: %w", err)
	}

	if err := os.WriteFile(filepath.Join(tempDir, "user-data"), userData.Bytes(), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tempDir, "vendor-data"), []byte(vendorDataTemplate), 0o644); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(tempDir, "meta-data"), []byte("instance-id: "+s.Profile), 0o644); err != nil {
		return "", err
	}

	isoPath := s.GetSeedISOPath()
	_ = os.MkdirAll(filepath.Dir(isoPath), 0o755)

	cmd := exec.Command("xorrisofs", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock",
		filepath.Join(tempDir, "user-data"), filepath.Join(tempDir, "meta-data"), filepath.Join(tempDir, "vendor-data"))
	if err := cmd.Run(); err != nil {
		return "", err
	}

	return isoPath, nil
}

func VerifyDockerHealth(s *config.State) bool {
	httpClient := http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var dialer net.Dialer
				return dialer.DialContext(ctx, "unix", s.GetDockerSocketPath())
			},
		},
	}

	err := retry.WithTimeout(context.Background(), 2*time.Minute, time.Second, 5*time.Second, func() error {
		resp, err := httpClient.Get("http://localhost/_ping")
		if err == nil && resp.StatusCode == http.StatusOK {
			return nil
		}
		fmt.Printf("... checking Docker health: %v\n", err)
		return err
	})

	return err == nil
}

func OrchestrateEnvironment(ctx context.Context, s *config.State) error {
	fmt.Println("⏳ Synchronizing network handshake channels...")

	client, err := ssh.GetClient(ctx, s)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer func() { _ = client.Close() }()

	if s.Cfg.AlpineSetup.Bootstrap == "tiny" {
		fmt.Println("⏳ Waiting for tiny-cloud...")
		if err := ssh.RunCommand(client, `while [[ $(tiny-cloud --bootstrap status) != "complete" ]]; do sleep 1; done`); err != nil {
			fmt.Printf("⚠️ Warning: tiny-cloud status check failed: %v\n", err)
		}
	} else {
		fmt.Println("⏳ Waiting for cloud-init...")
		if err := ssh.RunCommand(client, "cloud-init status --wait"); err != nil {
			fmt.Printf("⚠️ Warning: cloud-init status check failed: %v\n", err)
		}
	}

	if len(s.Cfg.Provision.Commands) != 0 {
		provisioned := false
		if err := ssh.RunCommand(client, "doas test -f /.provisioned"); err == nil {
			provisioned = true
		}

		if !provisioned {
			fmt.Println("🔐 Injecting runtime framework modules...")
			for _, command := range s.Cfg.Provision.Commands {
				if err := ssh.RunCommand(client, command); err != nil {
					fmt.Printf("⚠️ Provisioning command failed: %s (%v)\n", command, err)
				}
			}
			if err := ssh.RunCommand(client, "doas touch /.provisioned"); err != nil {
				fmt.Printf("⚠️ Failed to mark VM as provisioned: %v\n", err)
			}
		} else {
			fmt.Println("ℹ️ VM already provisioned. Skipping initial setup steps.")
		}
	}

	fmt.Println("📁 Syncing remote storage partition mounts...")
	for i, m := range s.Cfg.Mounts {
		tag := fmt.Sprintf("mount%d", i)
		if err := ssh.RunCommand(client, fmt.Sprintf("doas mkdir -p %s", m)); err != nil {
			fmt.Printf("⚠️ Failed to create mount directory %s: %v\n", m, err)
			continue
		}
		if err := ssh.RunCommand(client, fmt.Sprintf("doas chown -R %s:docker %s", s.Cfg.VM.SSHUser, m)); err != nil {
			fmt.Printf("⚠️ Failed to set ownership on %s: %v\n", m, err)
		}
		if err := ssh.RunCommand(client, fmt.Sprintf("doas chmod 775 %s", m)); err != nil {
			fmt.Printf("⚠️ Failed to set permissions on %s: %v\n", m, err)
		}

		// Check if already mounted to avoid errors
		checkCmd := fmt.Sprintf("mount | grep -q 'on %s type 9p'", m)
		if err := ssh.RunCommand(client, checkCmd); err != nil {
			mountCmd := fmt.Sprintf("doas mount -t 9p -o trans=virtio,version=9p2000.L,_netdev,rw,access=any %s %s", tag, m)
			if err := ssh.RunCommand(client, mountCmd); err != nil {
				fmt.Printf("⚠️ Mount failed for %s: %v\n", m, err)
			}
		}
	}

	fmt.Println("⏳ Waiting for Docker...")
	// Wait for up to 60 seconds for the docker socket to exist
	waitDockerCmd := "timeout 60 sh -c 'until [ -S /var/run/docker.sock ]; do sleep 1; done'"
	if err := ssh.RunCommand(client, waitDockerCmd); err != nil {
		fmt.Printf("⚠️ Warning: Docker socket not found after : %v\n", err)
	}

	fmt.Println("🚀 Orchestration complete. Spawning background tunnel...")
	return nil
}
