package profiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/luisdavim/termux-docker/pkg/config"
	"github.com/luisdavim/termux-docker/pkg/vm"
)

func Delete(state *config.State) error {
	if err := vm.Stop(state); err != nil {
		fmt.Printf("⚠️ Warning: error during stop phase: %v\n", err)
	}

	// Delete disk image
	if state.Cfg.VM.DiskPath != "" {
		if _, err := os.Stat(state.Cfg.VM.DiskPath); err == nil {
			fmt.Printf("🗑️ Deleting disk image: %s\n", state.Cfg.VM.DiskPath)
			if err := os.Remove(state.Cfg.VM.DiskPath); err != nil {
				fmt.Printf("❌ Failed to delete disk image: %v\n", err)
			}
		}
	}

	// Delete config file
	configFile := config.GetConfigFilename(state.Profile, state.HomeDir)
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("🗑️ Deleting config file: %s\n", configFile)
		if err := os.Remove(configFile); err != nil {
			fmt.Printf("❌ Failed to delete config file: %v\n", err)
		}
	}

	// Delete seed ISO
	seedISO := state.GetSeedISOPath()
	if _, err := os.Stat(seedISO); err == nil {
		fmt.Printf("🗑️ Deleting seed ISO: %s\n", seedISO)
		if err := os.Remove(seedISO); err != nil {
			fmt.Printf("❌ Failed to delete seed ISO: %v\n", err)
		}
	}

	// Delete QEMU log
	qemuLog := state.GetLogPath()
	if _, err := os.Stat(qemuLog); err == nil {
		fmt.Printf("🗑️ Deleting QEMU log: %s\n", qemuLog)
		if err := os.Remove(qemuLog); err != nil {
			fmt.Printf("❌ Failed to delete QEMU log: %v\n", err)
		}
	}

	// Delete Tunnel log
	tunnelLog := state.GetTunnelLogPath()
	if _, err := os.Stat(tunnelLog); err == nil {
		fmt.Printf("🗑️ Deleting Tunnel log: %s\n", tunnelLog)
		if err := os.Remove(tunnelLog); err != nil {
			fmt.Printf("❌ Failed to delete tunnel log: %v\n", err)
		}
	}

	// Clean up empty socket directory for non-default profiles
	if state.Profile != "default" && state.Profile != "" {
		socketDir := filepath.Join(state.HomeDir, ".docker", state.Profile)
		if entries, err := os.ReadDir(socketDir); err == nil && len(entries) == 0 {
			if err := os.Remove(socketDir); err != nil {
				fmt.Printf("❌ Failed to remove empty socket directory: %v\n", err)
			}
		}
	}

	fmt.Printf("✅ Profile '%s' successfully deleted.\n", state.Profile)
	return nil
}
