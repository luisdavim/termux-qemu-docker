package config

import (
	"crypto/rand"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/luisdavim/termux-qemu-docker/pkg/utils"
)

func (c *Config) Validate() error {
	if _, err := strconv.Atoi(c.VM.Memory); err != nil {
		return fmt.Errorf("invalid memory configuration: %s. Must be numeric (MB)", c.VM.Memory)
	}

	if c.AlpineSetup.Bootstrap != "tiny" && c.AlpineSetup.Bootstrap != "cloudinit" {
		return fmt.Errorf("invalid Bootstrap configuration: %s. Muat be tiny or cloudinit", c.AlpineSetup.Bootstrap)
	}

	return nil
}

func (c *Config) SetDefaults(profile, homeDir, prefix string) {
	if c.AlpineSetup.Mirror == "" {
		c.AlpineSetup.Mirror = "https://dl-cdn.alpinelinux.org/alpine"
	}

	if c.AlpineSetup.Arch == "" {
		c.AlpineSetup.Arch = "aarch64"
		if runtime.GOARCH == "amd64" {
			c.AlpineSetup.Arch = "x86_64"
		}
	}

	if c.AlpineSetup.Version == "latest" {
		if v, err := utils.GetLatestAlpineVersion(c.AlpineSetup.Mirror, c.AlpineSetup.Arch); err == nil {
			c.AlpineSetup.Version = v
		} else {
			c.AlpineSetup.Version = ""
		}
	}

	if c.AlpineSetup.Version == "" {
		c.AlpineSetup.Version = "3.23.4"
	}

	if c.AlpineSetup.Timezone == "" {
		c.AlpineSetup.Timezone = "UTC"
	}

	if c.AlpineSetup.Bootstrap == "" {
		c.AlpineSetup.Bootstrap = "tiny"
	}

	if c.VM.CPUs == 0 {
		c.VM.CPUs = utils.NumCPU() / 2
		if c.VM.CPUs == 0 {
			c.VM.CPUs = 2
		}
	}
	if c.VM.Memory == "" {
		c.VM.Memory = "2048"
	}
	if c.VM.DiskSizeGB == 0 {
		c.VM.DiskSizeGB = 10
	}

	if c.VM.SSHUser == "" {
		c.VM.SSHUser = "termux"
	}
	if c.VM.SSHPassword == "" {
		c.VM.SSHPassword = rand.Text()
	}

	if c.VM.SSHPort == 0 {
		if profile == "default" || profile == "" {
			c.VM.SSHPort = 2222
		} else {
			h := fnv.New32a()
			h.Write([]byte(profile))
			portOffset := int(h.Sum32() % 500)
			c.VM.SSHPort = 2222 + portOffset
		}
	}

	if c.VM.BiosPath == "" {
		c.VM.BiosPath = fmt.Sprintf(filepath.Join(prefix, "/share/qemu/edk2-%s-code.fd"), c.AlpineSetup.Arch)
	}

	if c.VM.BiosVarsPath == "" {
		arch := c.AlpineSetup.Arch
		if arch == "aarch64" {
			arch = "arm"
		}
		if arch == "x86_64" {
			arch = "i386"
		}
		c.VM.BiosVarsPath = fmt.Sprintf(filepath.Join(prefix, "/share/qemu/edk2-%s-vars.fd"), arch)
	}

	if c.VM.DiskPath == "" {
		configDir := GetBaseDir(homeDir)
		if profile == "default" || profile == "" {
			c.VM.DiskPath = filepath.Join(configDir, "alpine.img")
		} else {
			c.VM.DiskPath = filepath.Join(configDir, fmt.Sprintf("alpine-%s.img", profile))
		}
	}

	if len(c.Mounts) == 0 {
		c.Mounts = []string{filepath.Join(homeDir)}
		tmpDir := os.Getenv("TMPDIR")
		if tmpDir == "" {
			tmpDir = filepath.Join(prefix, "/tmp")
		}
		c.Mounts = append(c.Mounts, tmpDir)
	}
}
