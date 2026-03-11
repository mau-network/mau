// Package config implements application configuration management.
// This package is GTK-agnostic and contains pure configuration logic.
package config

// AppConfig stores application preferences
type AppConfig struct {
	SchemaVersion   int           `json:"schemaVersion"` // Config schema version for migrations
	DarkMode        bool          `json:"darkMode"`
	AutoSync        bool          `json:"autoSync"`
	AutoSyncMinutes int           `json:"autoSyncMinutes"`
	ServerPort      int           `json:"serverPort"` // Configurable server port (default: 8080)
	LastAccount     string        `json:"lastAccount"`
	Accounts        []AccountInfo `json:"accounts"`
}

const CurrentSchemaVersion = 1

// AccountInfo stores account metadata
type AccountInfo struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Fingerprint string `json:"fingerprint"`
	DataDir     string `json:"dataDir"`
}

// DefaultConfig returns default configuration
func DefaultConfig() AppConfig {
	return AppConfig{
		SchemaVersion:   CurrentSchemaVersion,
		DarkMode:        false,
		AutoSync:        false,
		AutoSyncMinutes: 30,
		ServerPort:      8080,
		Accounts:        []AccountInfo{},
	}
}

// Migrate performs schema migrations
func (cfg *AppConfig) Migrate() {
	if cfg.SchemaVersion == 0 {
		// Migrate from v0 to v1
		cfg.SchemaVersion = 1
		if cfg.ServerPort == 0 {
			cfg.ServerPort = 8080
		}
	}
	// Future migrations go here
}
