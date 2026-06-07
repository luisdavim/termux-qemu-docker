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

func (s *State) getPath(filename string) string {
	configDir := GetBaseDir(s.HomeDir)
	if s.Profile == "" {
		s.Profile = "default"
	}
	return filepath.Join(configDir, filename)
}

func (s *State) GetSSHHostKeyFile() string {
	return s.getPath(fmt.Sprintf("%s.pub", s.Profile))
}

func (s *State) GetSeedISOPath() string {
	return s.getPath(fmt.Sprintf("seed-%s.iso", s.Profile))
}

func (s *State) GetBootVarsPath() string {
	return s.getPath(fmt.Sprintf("boot-vars-%s.fd", s.Profile))
}

func (s *State) GetPIDFile() string {
	return s.getPath(fmt.Sprintf("vm-%s.pid", s.Profile))
}

func (s *State) GetTunnelPIDFile() string {
	return s.getPath(fmt.Sprintf("tunnel-%s.pid", s.Profile))
}

func (s *State) GetTunnelLogPath() string {
	return s.getPath(fmt.Sprintf("tunnel-%s.log", s.Profile))
}

func (s *State) GetLogPath() string {
	return s.getPath(fmt.Sprintf("qemu-%s.log", s.Profile))
}

func (s *State) GetDockerSocketPath() string {
	return s.getPath(fmt.Sprintf("docker-%s.sock", s.Profile))
}

func (s *State) GetPortMapFile() string {
	return s.getPath(fmt.Sprintf("ports-%s.json", s.Profile))
}

func (s *State) UpdateArch(arch string) error {
	s.Cfg.AlpineSetup.Arch = arch
	s.Cfg.VM.BiosPath = fmt.Sprintf(filepath.Join(s.Prefix, "/share/qemu/edk2-%s-code.fd"), s.Cfg.AlpineSetup.Arch)
	if arch == "aarch64" {
		arch = "arm"
	}
	if arch == "x86_64" {
		arch = "i386"
	}
	s.Cfg.VM.BiosVarsPath = fmt.Sprintf(filepath.Join(s.Prefix, "/share/qemu/edk2-%s-vars.fd"), arch)
	return nil
}
