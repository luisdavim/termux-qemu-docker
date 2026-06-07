package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VM struct {
		CPUs         int    `yaml:"cpus"`
		Memory       string `yaml:"memory"`
		BiosPath     string `yaml:"bios"`
		BiosVarsPath string `yaml:"bios_vars"`
		DiskPath     string `yaml:"disk_path"`
		DiskSizeGB   int    `yaml:"disk_size_gb"`
		SSHPort      int    `yaml:"ssh_port"`
		SSHUser      string `yaml:"ssh_user"`
		SSHPassword  string `yaml:"ssh_password"`
	} `yaml:"vm"`

	AlpineSetup struct {
		Version   string `yaml:"version"`
		Mirror    string `yaml:"mirror"`
		Arch      string `yaml:"arch"`
		Bootstrap string `yaml:"bootstrap"`
		Timezone  string `yaml:"timezone"`
	} `yaml:"alpine_setup"`

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

func GetBaseDir(homeDir string) string {
	return filepath.Join(homeDir, ".termux-qemu-docker")
}

func GetConfigFilename(profile string, homeDir string) string {
	configDir := GetBaseDir(homeDir)

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

	_ = os.MkdirAll(filepath.Dir(filename), 0o755)
	return os.WriteFile(filename, data, 0o644)
}
