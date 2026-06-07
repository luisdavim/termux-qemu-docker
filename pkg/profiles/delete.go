package profiles

import (
	"fmt"
	"os"

	"github.com/luisdavim/termux-qemu-docker/pkg/config"
	"github.com/luisdavim/termux-qemu-docker/pkg/vm"
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

	// Delete boot vars
	bootVars := state.GetBootVarsPath()
	if _, err := os.Stat(bootVars); err == nil {
		fmt.Printf("🗑️ Deleting boot vars: %s\n", bootVars)
		if err := os.Remove(bootVars); err != nil {
			fmt.Printf("❌ Failed to delete boot vars: %v\n", err)
		}
	}

	// Delete SSH hosk key
	hostKey := state.GetSSHHostKeyFile()
	if _, err := os.Stat(hostKey); err == nil {
		fmt.Printf("🗑️ Deleting SSH host key: %s\n", hostKey)
		if err := os.Remove(hostKey); err != nil {
			fmt.Printf("❌ Failed to delete SSH host key: %v\n", err)
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

	// Delete Port Map
	portMap := state.GetPortMapFile()
	if _, err := os.Stat(portMap); err == nil {
		fmt.Printf("🗑️ Deleting port map: %s\n", portMap)
		if err := os.Remove(portMap); err != nil {
			fmt.Printf("❌ Failed to delete port map: %v\n", err)
		}
	}

	fmt.Printf("✅ Profile '%s' successfully deleted.\n", state.Profile)
	return nil
}
