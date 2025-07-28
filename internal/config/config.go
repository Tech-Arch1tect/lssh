package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tech-arch1tect/lssh/internal/provider"
)

type Config struct {
	Providers      []provider.Config `json:"providers"`
	CacheEnabled   *bool             `json:"cache_enabled,omitempty"`
	ExcludeGroups  []string          `json:"exclude_groups,omitempty"`
	ExcludeHosts   []string          `json:"exclude_hosts,omitempty"`
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
	providerType := getProviderType(hostsFile)

	return &Config{
		Providers: []provider.Config{
			{
				Type: providerType,
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

func getProviderType(filePath string) string {
	if envType := os.Getenv("LSSH_PROVIDER_TYPE"); envType != "" {
		return envType
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yml", ".yaml":
		return "ansible"
	case ".json":
		return "json"
	default:
		return "json"
	}
}

func (c *Config) IsCacheEnabled() bool {
	if envValue := os.Getenv("LSSH_CACHE_ENABLED"); envValue != "" {
		if enabled, err := strconv.ParseBool(envValue); err == nil {
			return enabled
		}
	}

	if c.CacheEnabled != nil {
		return *c.CacheEnabled
	}

	return true
}

func (c *Config) GetExcludeGroups() []string {
	if envValue := os.Getenv("LSSH_EXCLUDE_GROUPS"); envValue != "" {
		return strings.Split(envValue, ",")
	}
	return c.ExcludeGroups
}

func (c *Config) GetExcludeHosts() []string {
	if envValue := os.Getenv("LSSH_EXCLUDE_HOSTS"); envValue != "" {
		return strings.Split(envValue, ",")
	}
	return c.ExcludeHosts
}

func matchesPattern(name, pattern string) bool {
	if pattern == "" {
		return false
	}
	
	if !strings.Contains(pattern, "*") {
		return name == pattern
	}
	
	parts := strings.Split(pattern, "*")
	if len(parts) == 0 {
		return false
	}
	
	pos := 0
	for i, part := range parts {
		if part == "" {
			continue
		}
		
		index := strings.Index(name[pos:], part)
		if index == -1 {
			return false
		}
		
		if i == 0 && index != 0 {
			return false
		}
		
		pos += index + len(part)
	}
	
	if len(parts) > 0 && parts[len(parts)-1] != "" && !strings.HasSuffix(name, parts[len(parts)-1]) {
		return false
	}
	
	return true
}

func (c *Config) IsGroupExcluded(groupName string) bool {
	for _, pattern := range c.GetExcludeGroups() {
		if matchesPattern(groupName, strings.TrimSpace(pattern)) {
			return true
		}
	}
	return false
}

func (c *Config) IsHostExcluded(hostName string) bool {
	for _, pattern := range c.GetExcludeHosts() {
		if matchesPattern(hostName, strings.TrimSpace(pattern)) {
			return true
		}
	}
	return false
}
