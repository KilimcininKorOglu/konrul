// Konrul - Configuration file support
// Cross-platform config file locations

package main

import (
	"os"
	"path/filepath"
	"runtime"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	RefreshInterval int    `yaml:"refresh_interval"` // seconds (1-10)
	DefaultSort     string `yaml:"default_sort"`     // cpu, mem, pid, name
	TreeView        bool   `yaml:"tree_view"`        // start in tree view
	Theme           string `yaml:"theme"`            // default, dark, light (future)
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		RefreshInterval: 1,
		DefaultSort:     "cpu",
		TreeView:        false,
		Theme:           "default",
	}
}

// getConfigPath returns the config file path based on OS
func getConfigPath() string {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// %APPDATA%\konrul\config.yaml
		configDir = os.Getenv("APPDATA")
		if configDir == "" {
			configDir = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
	default:
		// ~/.config/konrul/config.yaml (Linux, macOS, FreeBSD)
		configDir = os.Getenv("XDG_CONFIG_HOME")
		if configDir == "" {
			home, _ := os.UserHomeDir()
			configDir = filepath.Join(home, ".config")
		}
	}

	return filepath.Join(configDir, "konrul", "config.yaml")
}

// LoadConfig loads configuration from file or returns defaults.
// If customPath is non-empty, it is used instead of the default path.
func LoadConfig(customPath string) Config {
	config := DefaultConfig()

	configPath := customPath
	if configPath == "" {
		configPath = getConfigPath()
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config file doesn't exist, use defaults
		return config
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		// Invalid config, use defaults
		return DefaultConfig()
	}

	// Validate values
	if config.RefreshInterval < 1 {
		config.RefreshInterval = 1
	} else if config.RefreshInterval > 10 {
		config.RefreshInterval = 10
	}

	// Validate sort mode
	switch config.DefaultSort {
	case "cpu", "mem", "memory", "pid", "name":
		// valid
	default:
		config.DefaultSort = "cpu"
	}

	return config
}

// SaveConfig saves configuration to file
func SaveConfig(config Config) error {
	configPath := getConfigPath()

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(&config)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// CreateDefaultConfigFile creates a default config file if it doesn't exist
func CreateDefaultConfigFile() error {
	configPath := getConfigPath()

	// Check if file already exists
	if _, err := os.Stat(configPath); err == nil {
		return nil // File exists
	}

	return SaveConfig(DefaultConfig())
}
