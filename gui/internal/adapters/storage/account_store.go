package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mau-network/mau"
)

// AccountStore implements account.Store using Mau's PGP-based account system
type AccountStore struct {
	dataDir  string
	password string // Demo password (in production, this would come from user input)
}

// NewAccountStore creates a new account store
func NewAccountStore(dataDir string) *AccountStore {
	return &AccountStore{
		dataDir:  dataDir,
		password: "demo", // TODO: Remove hardcoded password in production
	}
}

// Init initializes or loads the account
func (s *AccountStore) Init() (*mau.Account, error) {
	if err := os.MkdirAll(s.dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	accountPath := filepath.Join(s.dataDir, ".mau", "account.pgp")

	if _, err := os.Stat(accountPath); os.IsNotExist(err) {
		return s.Create("Demo User", "demo@mau.network", s.password)
	}

	return s.Open(s.password)
}

// Create creates a new account
func (s *AccountStore) Create(name, email, password string) (*mau.Account, error) {
	acc, err := mau.NewAccount(s.dataDir, name, email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return acc, nil
}

// Open opens an existing account
func (s *AccountStore) Open(password string) (*mau.Account, error) {
	acc, err := mau.OpenAccount(s.dataDir, password)
	if err != nil {
		return nil, fmt.Errorf("failed to open account: %w", err)
	}

	return acc, nil
}

// DataDir returns the data directory
func (s *AccountStore) DataDir() string {
	return s.dataDir
}
