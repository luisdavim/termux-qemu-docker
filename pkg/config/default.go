package config

import (
	"fmt"
	"hash/fnv"
	"path/filepath"
	"runtime"
)

func (c *Config) SetDefaults(profile, homeDir, prefix string) {
	if c.VM.CPUs == 0 {
		c.VM.CPUs = 2
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
		c.VM.SSHPassword = "password123"
	}

	if c.VM.DiskPath == "" {
		configDir := GetBaseDir(homeDir)
		if profile == "default" || profile == "" {
			c.VM.DiskPath = filepath.Join(configDir, "alpine.img")
		} else {
			c.VM.DiskPath = filepath.Join(configDir, fmt.Sprintf("alpine-%s.img", profile))
		}
	}

	if c.VM.SSHPort == 0 {
		if profile == "default" || profile == "" {
			c.VM.SSHPort = 2222
		} else {
			h := fnv.New32a()
			h.Write([]byte(profile))
			portOffset := int(h.Sum32() % 500)
			c.VM.SSHPort = 2222 + portOffset
			if c.VM.DockerPort == 0 {
				c.VM.DockerPort = 2375 + portOffset
			}
		}

		if c.VM.DockerPort == 0 {
			c.VM.DockerPort = 2375
		}
	}

	if c.AlpineSetup.Version == "" {
		c.AlpineSetup.Version = "3.23.4"
	}
	if c.AlpineSetup.Arch == "" {
		c.AlpineSetup.Arch = "aarch64"
		if runtime.GOARCH == "amd64" {
			c.AlpineSetup.Arch = "x86_64"
		}
	}
	if c.AlpineSetup.Mirror == "" {
		c.AlpineSetup.Mirror = "https://dl-cdn.alpinelinux.org/alpine"
	}
	if c.AlpineSetup.Timezone == "" {
		c.AlpineSetup.Timezone = "UTC"
	}

	if c.Termux.SSHPort == 0 {
		c.Termux.SSHPort = 8022
	}

	if len(c.Mounts) == 0 {
		c.Mounts = []string{filepath.Join(homeDir)}
		c.Mounts = append(c.Mounts, filepath.Join(prefix, "/tmp"))
	}
}
