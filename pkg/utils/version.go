package utils

import (
	"fmt"
	"io"
	"net/http"

	"gopkg.in/yaml.v3"
)

type AlpineRelease struct {
	Version string `yaml:"version"`
}

func GetLatestAlpineVersion(mirror, arch string) (string, error) {
	url := fmt.Sprintf("%s/latest-stable/releases/%s/latest-releases.yaml", mirror, arch)
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest releases metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch metadata, status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read metadata body: %w", err)
	}

	var releases []AlpineRelease
	if err := yaml.Unmarshal(body, &releases); err != nil {
		return "", fmt.Errorf("failed to parse latest releases YAML: %w", err)
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found in metadata")
	}

	return releases[0].Version, nil
}
