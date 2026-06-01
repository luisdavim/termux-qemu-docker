package config

import (
	"encoding/json"
	"os"
)

type PortMapping struct {
	LocalAddress string `json:"local_address"`
	VMAddress    string `json:"vm_address"`
	Status       string `json:"status"`
}

type PortMapState struct {
	Mappings []PortMapping `json:"mappings"`
}

func SavePortMappings(path string, state PortMapState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func LoadPortMappings(path string) (PortMapState, error) {
	var state PortMapState
	data, err := os.ReadFile(path)
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}
