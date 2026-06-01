package config

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VM struct {
		CPUs        int    `yaml:"cpus"`
		Memory      string `yaml:"memory"`
		DiskPath    string `yaml:"disk_path"`
		DiskSizeGB  int    `yaml:"disk_size_gb"`
		SSHPort     int    `yaml:"ssh_port"`
		DockerPort  int    `yaml:"docker_port"`
		SSHUser     string `yaml:"ssh_user"`
		SSHPassword string `yaml:"ssh_password"`
	} `yaml:"vm"`

	AlpineSetup struct {
		Version  string `yaml:"version"`
		Mirror   string `yaml:"mirror"`
		Arch     string `yaml:"arch"`
		Timezone string `yaml:"timezone"`
	} `yaml:"alpine_setup"`

	Termux struct {
		SSHUser string `yaml:"ssh_user"`
		SSHPort int    `yaml:"ssh_port"`
	} `yaml:"termux"`

	Mounts []string `yaml:"mounts"`

	Provision struct {
		Commands []string `yaml:"commands"`
	} `yaml:"provision"`
}

func NewDefaultConfig(profile, homeDir, prefix string) *Config {
	c := &Config{}
	c.SetDefaults(profile, homeDir, prefix)
	return c
}

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
		c.AlpineSetup.Version = "3.21.7"
	}
	if c.AlpineSetup.Arch == "" {
		c.AlpineSetup.Arch = "aarch64"
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

func GetBaseDir(homeDir string) string {
	return filepath.Join(homeDir, ".termux-docker")
}

func GetConfigFilename(profile string, homeDir string) string {
	configDir := GetBaseDir(homeDir)
	_ = os.MkdirAll(configDir, 0o755)

	filename := "config.yaml"
	if profile != "default" && profile != "" {
		filename = fmt.Sprintf("config-%s.yaml", profile)
	}
	return filepath.Join(configDir, filename)
}

func LoadConfig(profile string, homeDir string) (*Config, error) {
	filename := GetConfigFilename(profile, homeDir)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	var c Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&c); err != nil {
		return nil, fmt.Errorf("failed to decode config file %s: %w", filename, err)
	}
	return &c, nil
}

func SaveConfig(profile string, homeDir string, c *Config) error {
	filename := GetConfigFilename(profile, homeDir)

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(filename, data, 0o644)
}
