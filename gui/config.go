package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/mau-network/mau"
)

// AppConfig stores application preferences
type AppConfig struct {
	SchemaVersion   int           `json:"schemaVersion"` // Config schema version for migrations
	DarkMode        bool          `json:"darkMode"`
	AutoStartServer bool          `json:"autoStartServer"`
	AutoSync        bool          `json:"autoSync"`
	AutoSyncMinutes int           `json:"autoSyncMinutes"`
	ServerPort      int           `json:"serverPort"` // Configurable server port (default: 8080)
	LastAccount     string        `json:"lastAccount"`
	Accounts        []AccountInfo `json:"accounts"`
}

const currentSchemaVersion = 1

// AccountInfo stores account metadata
type AccountInfo struct {
	Name        string `json:"name"`
	Email       string `json:"email"`
	Fingerprint string `json:"fingerprint"`
	DataDir     string `json:"dataDir"`
}

// ConfigManager handles config persistence
type ConfigManager struct {
	configPath string
	config     AppConfig
}

// NewConfigManager creates a config manager
func NewConfigManager(dataDir string) *ConfigManager {
	cm := &ConfigManager{
		configPath: filepath.Join(dataDir, configFile),
		config: AppConfig{
			SchemaVersion:   currentSchemaVersion,
			DarkMode:        false,
			AutoStartServer: false,
			AutoSync:        false,
			AutoSyncMinutes: 30,
			ServerPort:      8080,
			Accounts:        []AccountInfo{},
		},
	}
	cm.Load()
	return cm
}

// Load reads config from disk
func (cm *ConfigManager) Load() error {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		// File doesn't exist - use defaults
		return nil
	}

	if err := json.Unmarshal(data, &cm.config); err != nil {
		return err
	}

	// Migrate old configs
	if cm.config.SchemaVersion == 0 {
		// Migrate from v0 to v1
		cm.config.SchemaVersion = 1
		if cm.config.ServerPort == 0 {
			cm.config.ServerPort = 8080
		}
		// Save migrated config
		cm.Save()
	}

	return nil
}

// Save writes config to disk atomically
func (cm *ConfigManager) Save() error {
	data, err := json.MarshalIndent(cm.config, "", "  ")
	if err != nil {
		return err
	}

	// Atomic write: write to temp file, then rename
	tmpPath := cm.configPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}

	// Rename is atomic on POSIX systems
	return os.Rename(tmpPath, cm.configPath)
}

// Get returns current config
func (cm *ConfigManager) Get() AppConfig {
	return cm.config
}

// Update modifies config and saves
func (cm *ConfigManager) Update(updater func(*AppConfig)) error {
	updater(&cm.config)
	return cm.Save()
}

// AccountManager handles account operations
type AccountManager struct {
	dataDir  string
	account  *mau.Account
	password string
}

// NewAccountManager creates an account manager
func NewAccountManager(dataDir string) *AccountManager {
	return &AccountManager{
		dataDir: dataDir,
	}
}

// Init initializes or loads the account
func (am *AccountManager) Init() error {
	if err := os.MkdirAll(am.dataDir, 0700); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	accountPath := filepath.Join(am.dataDir, ".mau", "account.pgp")

	if _, err := os.Stat(accountPath); os.IsNotExist(err) {
		return am.createAccount()
	}

	return am.loadAccount()
}

func (am *AccountManager) createAccount() error {
	name, email, password := "Demo User", "demo@mau.network", "demo"

	acc, err := mau.NewAccount(am.dataDir, name, email, password)
	if err != nil {
		return fmt.Errorf("failed to create account: %w", err)
	}

	am.account = acc
	am.password = password
	return nil
}

func (am *AccountManager) loadAccount() error {
	password := "demo"

	acc, err := mau.OpenAccount(am.dataDir, password)
	if err != nil {
		return fmt.Errorf("failed to load account: %w", err)
	}

	am.account = acc
	am.password = password
	return nil
}

// Account returns the current account
func (am *AccountManager) Account() *mau.Account {
	return am.account
}

// Info returns account info for config
func (am *AccountManager) Info() AccountInfo {
	return AccountInfo{
		Name:        am.account.Name(),
		Email:       am.account.Email(),
		Fingerprint: am.account.Fingerprint().String(),
		DataDir:     am.dataDir,
	}
}

// ApplyTheme applies dark/light theme
func ApplyTheme(app *adw.Application, darkMode bool) {
	if darkMode {
		app.StyleManager().SetColorScheme(adw.ColorSchemeForceDark)
	} else {
		app.StyleManager().SetColorScheme(adw.ColorSchemeForceLight)
	}
}
