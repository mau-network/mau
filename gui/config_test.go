package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigManager_NewConfigManager(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)

	if cm == nil {
		t.Fatal("NewConfigManager returned nil")
	}

	cfg := cm.Get()
	if cfg.DarkMode != false {
		t.Errorf("Expected DarkMode=false, got %v", cfg.DarkMode)
	}
	if cfg.AutoSyncMinutes != 30 {
		t.Errorf("Expected AutoSyncMinutes=30, got %d", cfg.AutoSyncMinutes)
	}
}

func TestConfigManager_SaveLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)

	// Update config
	err := cm.Update(func(cfg *AppConfig) {
		cfg.DarkMode = true
		cfg.AutoStartServer = true
		cfg.AutoSyncMinutes = 60
	})
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	configPath := filepath.Join(tmpDir, configFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load in new instance
	cm2 := NewConfigManager(tmpDir)
	cfg := cm2.Get()

	if !cfg.DarkMode {
		t.Error("DarkMode not persisted")
	}
	if !cfg.AutoStartServer {
		t.Error("AutoStartServer not persisted")
	}
	if cfg.AutoSyncMinutes != 60 {
		t.Errorf("Expected AutoSyncMinutes=60, got %d", cfg.AutoSyncMinutes)
	}
}

func TestConfigManager_AddAccount(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)

	account := AccountInfo{
		Name:        "Test User",
		Email:       "test@example.com",
		Fingerprint: "ABC123",
		DataDir:     "/tmp/test",
	}

	err := cm.Update(func(cfg *AppConfig) {
		cfg.Accounts = append(cfg.Accounts, account)
		cfg.LastAccount = account.Fingerprint
	})
	if err != nil {
		t.Fatalf("Failed to add account: %v", err)
	}

	// Reload and verify
	cm2 := NewConfigManager(tmpDir)
	cfg := cm2.Get()

	if len(cfg.Accounts) != 1 {
		t.Fatalf("Expected 1 account, got %d", len(cfg.Accounts))
	}

	acc := cfg.Accounts[0]
	if acc.Name != "Test User" {
		t.Errorf("Expected name='Test User', got %s", acc.Name)
	}
	if acc.Email != "test@example.com" {
		t.Errorf("Expected email='test@example.com', got %s", acc.Email)
	}
	if cfg.LastAccount != "ABC123" {
		t.Errorf("Expected LastAccount='ABC123', got %s", cfg.LastAccount)
	}
}

func TestConfigManager_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, configFile)

	// Write invalid JSON
	err := os.WriteFile(configPath, []byte("{invalid json"), 0600)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	cm := NewConfigManager(tmpDir)
	// Should use defaults on invalid JSON
	cfg := cm.Get()

	if cfg.AutoSyncMinutes != 30 {
		t.Error("Should use default config on invalid JSON")
	}
}

func TestConfigManager_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)

	err := cm.Update(func(cfg *AppConfig) {
		cfg.DarkMode = true
	})
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	configPath := filepath.Join(tmpDir, configFile)
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Config file not found: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", perm)
	}
}

func TestConfigManager_ConcurrentAccess(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)

	// Multiple updates should not corrupt the file
	for i := 0; i < 10; i++ {
		val := i
		err := cm.Update(func(cfg *AppConfig) {
			cfg.AutoSyncMinutes = val
		})
		if err != nil {
			t.Fatalf("Update %d failed: %v", i, err)
		}
	}

	// Verify final value
	cfg := cm.Get()
	if cfg.AutoSyncMinutes != 9 {
		t.Errorf("Expected final value=9, got %d", cfg.AutoSyncMinutes)
	}

	// Verify JSON is valid
	configPath := filepath.Join(tmpDir, configFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config: %v", err)
	}

	var testCfg AppConfig
	if err := json.Unmarshal(data, &testCfg); err != nil {
		t.Fatalf("Config file corrupted: %v", err)
	}
}

func TestAccountInfo_JSON(t *testing.T) {
	info := AccountInfo{
		Name:        "Alice",
		Email:       "alice@example.com",
		Fingerprint: "FPR123",
		DataDir:     "/home/alice/.mau",
	}

	// Marshal
	data, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Unmarshal
	var decoded AccountInfo
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if decoded.Name != info.Name {
		t.Errorf("Name mismatch: expected %s, got %s", info.Name, decoded.Name)
	}
	if decoded.Email != info.Email {
		t.Errorf("Email mismatch: expected %s, got %s", info.Email, decoded.Email)
	}
	if decoded.Fingerprint != info.Fingerprint {
		t.Errorf("Fingerprint mismatch: expected %s, got %s", info.Fingerprint, decoded.Fingerprint)
	}
}

func TestAppConfig_Defaults(t *testing.T) {
	tmpDir := t.TempDir()
	cm := NewConfigManager(tmpDir)
	cfg := cm.Get()

	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"DarkMode", cfg.DarkMode, false},
		{"AutoStartServer", cfg.AutoStartServer, false},
		{"AutoSync", cfg.AutoSync, false},
		{"AutoSyncMinutes", cfg.AutoSyncMinutes, 30},
		{"LastAccount", cfg.LastAccount, ""},
		{"Accounts length", len(cfg.Accounts), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}
