package config

// Store defines the interface for configuration persistence
type Store interface {
	// Load reads configuration from storage
	Load() (AppConfig, error)

	// Save writes configuration to storage
	Save(config AppConfig) error
}
