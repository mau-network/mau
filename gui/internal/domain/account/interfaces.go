// Package account implements user account management.
// This package is GTK-agnostic and contains pure account logic.
package account

import "github.com/mau-network/mau"

// Info stores account metadata
type Info struct {
	Name        string
	Email       string
	Fingerprint string
	DataDir     string
}

// Store defines the interface for account persistence
type Store interface {
	// Init initializes or loads the account
	Init() (*mau.Account, error)

	// Create creates a new account
	Create(name, email, password string) (*mau.Account, error)

	// Open opens an existing account
	Open(password string) (*mau.Account, error)

	// DataDir returns the data directory for the account
	DataDir() string
}
