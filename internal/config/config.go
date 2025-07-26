package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tech-arch1tect/lssh/internal/provider"
)

type Config struct {
	Providers []provider.Config `json:"providers"`
}

func Load() (*Config, error) {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return getDefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func getConfigPath() string {
	if configDir := os.Getenv("XDG_CONFIG_HOME"); configDir != "" {
		return filepath.Join(configDir, "lssh", "config.json")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./config.json"
	}

	return filepath.Join(homeDir, ".config", "lssh", "config.json")
}

func getDefaultConfig() *Config {
	hostsFile := getDefaultHostsFile()
	return &Config{
		Providers: []provider.Config{
			{
				Type: "json",
				Name: "default",
				Config: map[string]interface{}{
					"file": hostsFile,
				},
			},
		},
	}
}

func getDefaultHostsFile() string {
	if envFile := os.Getenv("LSSH_HOSTS_FILE"); envFile != "" {
		return envFile
	}

	locations := []string{
		"./hosts.json",
		filepath.Join(os.Getenv("HOME"), ".config", "lssh", "hosts.json"),
		"/etc/lssh/hosts.json",
	}

	for _, location := range locations {
		if _, err := os.Stat(location); err == nil {
			return location
		}
	}

	return "./hosts.json"
}
