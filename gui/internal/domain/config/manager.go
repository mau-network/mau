package config

// Manager handles configuration operations
type Manager struct {
	store  Store
	config AppConfig
}

// NewManager creates a configuration manager
func NewManager(store Store) *Manager {
	m := &Manager{
		store:  store,
		config: DefaultConfig(),
	}
	m.Load()
	return m
}

// Load reads configuration from storage
func (m *Manager) Load() error {
	cfg, err := m.store.Load()
	if err != nil {
		// Use defaults if load fails
		return nil
	}

	// Perform any necessary migrations
	cfg.Migrate()

	m.config = cfg
	return m.store.Save(m.config) // Save if migrated
}

// Get returns current configuration (read-only)
func (m *Manager) Get() AppConfig {
	return m.config
}

// Update modifies configuration and persists it
func (m *Manager) Update(updater func(*AppConfig)) error {
	updater(&m.config)
	return m.store.Save(m.config)
}

// Save persists current configuration
func (m *Manager) Save() error {
	return m.store.Save(m.config)
}
