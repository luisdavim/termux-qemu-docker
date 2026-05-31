package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/tabwriter"

	"github.com/luisdavim/termux-docker/pkg/config"
)

func List(state *config.State) error {
	configDir := filepath.Join(state.HomeDir, ".termux-docker")
	_ = os.MkdirAll(configDir, 0o755)

	files, err := filepath.Glob(filepath.Join(configDir, "config*.yaml"))
	profiles := map[string]bool{"default": true}

	if err == nil {
		for _, f := range files {
			name := filepath.Base(f)
			if name == "config.yaml" {
				continue
			}
			// Ensure it matches config-<profile>.yaml and nothing else
			if !strings.HasPrefix(name, "config-") || !strings.HasSuffix(name, ".yaml") {
				continue
			}
			name = strings.TrimPrefix(name, "config-")
			name = strings.TrimSuffix(name, ".yaml")
			if name != "" {
				profiles[name] = true
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', 0)
	if _, err := fmt.Fprintln(w, "PROFILE\tSTATUS\tCPUS\tMEMORY\tDISK\tPID\tPORTS"); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}

	for p := range profiles {
		// Use a temporary state to get the correct PID file for each profile
		tempState := &config.State{
			Profile: p,
			HomeDir: state.HomeDir,
			Cfg:     state.Cfg,
		}
		pfile := tempState.GetPIDFile()

		status := "Offline"
		pidStr := "-"

		if data, err := os.ReadFile(pfile); err == nil {
			pid, _ := strconv.Atoi(string(data))
			process, err := os.FindProcess(pid)
			if err == nil && process.Signal(syscall.Signal(0)) == nil {
				status = "Running"
				pidStr = strconv.Itoa(pid)
			} else {
				_ = os.Remove(pfile)
			}
		}

		pCfg, err := config.LoadConfig(p, state.HomeDir)
		if err != nil {
			pCfg = config.NewDefaultConfig(p, state.HomeDir)
		} else {
			pCfg.SetDefaults(p, state.HomeDir)
		}

		portsStr := fmt.Sprintf("SSH:127.0.0.1:%d, Docker:127.0.0.1:%d", pCfg.VM.SSHPort, pCfg.VM.DockerPort)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%d\t%sMB\t%dGB\t%s\t%s\n",
			p, status, pCfg.VM.CPUs, pCfg.VM.Memory, pCfg.VM.DiskSizeGB, pidStr, portsStr); err != nil {
			return fmt.Errorf("failed to write profile %s: %w", p, err)
		}
	}
	return w.Flush()
}
