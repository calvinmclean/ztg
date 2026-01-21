package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v3"
)

// LoadFromFile loads configuration from a file (JSON/YAML)
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config

	// Determine format by file extension
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	case ".json":
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	default:
		// Default to JSON
		if err := json.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
	}

	return &cfg, nil
}

// SaveToFile saves configuration to a file (JSON format)
func SaveToFile(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LoadFromEnv loads environment variables into config using envconfig
func LoadFromEnv(cfg *Config) {
	if err := envconfig.Process("", cfg); err != nil {
		// Don't fail on env parsing, just use existing values
		// This allows partial environment overrides
	}
}
