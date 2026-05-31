package setup

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/luisdavim/termux-docker/pkg/config"
	"gopkg.in/yaml.v3"
)

func installPkgs(packages []string) error {
	fmt.Printf("📦 Updating system and installing packages: %s...\n", strings.Join(packages, ", "))

	args := append([]string{"install", "-y"}, packages...)
	pkgCmd := exec.Command("pkg", args...)
	pkgCmd.Stdout = os.Stdout
	pkgCmd.Stderr = os.Stderr

	if err := pkgCmd.Run(); err != nil {
		return fmt.Errorf("failed package requirements phase: %w", err)
	}

	return nil
}

func RunSetupEnvironment(homeDir string) error {
	fmt.Println("⚙️ Starting automated Termux dependency verification pipeline...")

	arch := "aarch64"
	if runtime.GOARCH == "amd64" {
		arch = "x86_64"
	}

	packages := []string{"qemu-utils", fmt.Sprintf("qemu-system-%s-headless", arch), "openssh", "libisoburn", "dosfstools", "docker"}
	if err := installPkgs(packages); err != nil {
		return err
	}
	fmt.Println("✅ Package requirements step satisfied.")

	biosPaths := []string{
		"/data/data/com.termux/files/usr/share/qemu/edk2-aarch64-code.fd",
		"/data/data/com.termux/files/usr/share/qemu/edk2-x86_64-code.fd",
	}

	biosFound := false
	for _, path := range biosPaths {
		if _, err := os.Stat(path); err == nil {
			biosFound = true
			break
		}
	}

	if !biosFound {
		fmt.Println("⚠️ QEMU EFI firmware blob layout not discovered in default standard locations.")
		fmt.Println("👉 Please ensure 'qemu-system-aarch64-headless' or 'qemu-system-x86_64-headless' completed setup configurations properly.")
	} else {
		fmt.Println("✅ QEMU boot components found.")
	}

	whoamiCmd := exec.Command("whoami")
	usernameBytes, err := whoamiCmd.Output()
	termuxUser := "termux"
	if err == nil {
		termuxUser = strings.TrimSpace(string(usernameBytes))
	}

	configFile := "config.yaml"
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		fmt.Println("📝 Default config.yaml absent. Materializing template framework mapping variables...")

		baseCfg := config.NewDefaultConfig("default", homeDir)
		baseCfg.Termux.SSHUser = termuxUser

		yamlData, err := yaml.Marshal(baseCfg)
		if err != nil {
			return fmt.Errorf("internal marshaling exception layer: %w", err)
		}

		if err := os.WriteFile(configFile, yamlData, 0o644); err != nil {
			return fmt.Errorf("configuration setup write file failure: %w", err)
		}
		fmt.Println("✅ Default config.yaml generated matching system profile contexts.")
	} else {
		fmt.Println("ℹ️ An active config.yaml already exists. Skipping fabrication overrides.")
	}

	fmt.Println("\n🎉 Setup sequence completed successfully!")
	fmt.Println("👉 Run your local OpenSSH background hook inside Termux next: 'sshd'")
	fmt.Println("🚀 Start your lightweight container virtualization layer: './termux-docker start'")
	return nil
}
