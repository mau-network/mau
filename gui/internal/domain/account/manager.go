package account

import (
	"fmt"

	"github.com/mau-network/mau"
)

// Manager handles account operations
type Manager struct {
	store           Store
	account         *mau.Account
	passphraseCache string // Session-only passphrase cache
}

// NewManager creates an account manager
func NewManager(store Store) *Manager {
	return &Manager{
		store: store,
	}
}

// Init initializes the account (create if needed, otherwise load)
func (m *Manager) Init() error {
	account, err := m.store.Init()
	if err != nil {
		return fmt.Errorf("account initialization failed: %w", err)
	}

	m.account = account
	return nil
}

// Account returns the current account
func (m *Manager) Account() *mau.Account {
	return m.account
}

// Info returns account metadata
func (m *Manager) Info() Info {
	return Info{
		Name:        m.account.Name(),
		Email:       m.account.Email(),
		Fingerprint: m.account.Fingerprint().String(),
	}
}

// CachePassphrase stores passphrase in memory for the session
func (m *Manager) CachePassphrase(passphrase string) {
	m.passphraseCache = passphrase
}

// GetCachedPassphrase retrieves the cached passphrase if available
func (m *Manager) GetCachedPassphrase() (string, bool) {
	if m.passphraseCache != "" {
		return m.passphraseCache, true
	}
	return "", false
}

// ClearPassphraseCache clears the in-memory passphrase cache
func (m *Manager) ClearPassphraseCache() {
	m.passphraseCache = ""
}
