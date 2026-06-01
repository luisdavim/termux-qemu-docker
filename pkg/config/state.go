package config

import (
	"fmt"
	"path/filepath"
)

type State struct {
	Profile    string
	HomeDir    string
	Prefix     string
	SSHHostKey []byte
	Cfg        *Config
}

func (s *State) GetSSHHostKeyFile() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("%s.pub", s.Profile))
}

func (s *State) GetSeedISOPath() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("seed-%s.iso", s.Profile))
}

func (s *State) GetPIDFile() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("vm-%s.pid", s.Profile))
}

func (s *State) GetTunnelPIDFile() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("tunnel-%s.pid", s.Profile))
}

func (s *State) GetTunnelLogPath() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("tunnel-%s.log", s.Profile))
}

func (s *State) GetLogPath() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("qemu-%s.log", s.Profile))
}

func (s *State) GetDockerSocketPath() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("docker-%s.sock", s.Profile))
}

func (s *State) GetPortMapFile() string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, fmt.Sprintf("ports-%s.json", s.Profile))
}

