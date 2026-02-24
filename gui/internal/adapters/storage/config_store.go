package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mau-network/mau/gui/internal/domain/config"
)

const configFileName = "config.json"

// ConfigStore implements config.Store using JSON file storage
type ConfigStore struct {
	configPath string
}

// NewConfigStore creates a new configuration store
func NewConfigStore(dataDir string) *ConfigStore {
	return &ConfigStore{
		configPath: filepath.Join(dataDir, configFileName),
	}
}

// Load reads configuration from disk
func (s *ConfigStore) Load() (config.AppConfig, error) {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist - return default config
			return config.DefaultConfig(), nil
		}
		return config.AppConfig{}, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return config.AppConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}

// Save writes configuration to disk atomically
func (s *ConfigStore) Save(cfg config.AppConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Atomic write: write to temp file, then rename
	tmpPath := s.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}

	// Rename is atomic on POSIX systems
	if err := os.Rename(tmpPath, s.configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}
