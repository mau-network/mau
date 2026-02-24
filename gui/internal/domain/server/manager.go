package server

import (
	"fmt"
	"log"
	"time"

	"github.com/mau-network/mau"
)

// Manager handles P2P server lifecycle
type Manager struct {
	account       *mau.Account
	server        *mau.Server
	running       bool
	statusHandler StatusCallback
}

// Config holds server manager configuration
type Config struct {
	Account       *mau.Account
	StatusHandler StatusCallback
}

// NewManager creates a server manager
func NewManager(cfg Config) *Manager {
	return &Manager{
		account:       cfg.Account,
		statusHandler: cfg.StatusHandler,
	}
}

// Start starts the P2P server
func (m *Manager) Start(port int) error {
	if m.running {
		return nil
	}

	server, err := m.account.Server(nil)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	m.server = server

	serverAddr := fmt.Sprintf(":%d", port)
	externalAddr := fmt.Sprintf("127.0.0.1:%d", port)

	// Start listener in background
	go func() {
		listener, err := mau.ListenTCP(serverAddr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", serverAddr, err)
			m.running = false
			m.server = nil
			if m.statusHandler != nil {
				m.statusHandler(false, serverAddr, err)
			}
			return
		}

		m.running = true
		if m.statusHandler != nil {
			m.statusHandler(true, serverAddr, nil)
		}

		// Serve blocks until error
		if err := m.server.Serve(listener, externalAddr); err != nil {
			log.Printf("Server error: %v", err)
			m.running = false
			if m.statusHandler != nil {
				m.statusHandler(false, serverAddr, err)
			}
		}
	}()

	// Wait a bit for startup to complete or fail
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop stops the P2P server
func (m *Manager) Stop() error {
	if !m.running || m.server == nil {
		return nil
	}

	if err := m.server.Close(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	m.running = false
	m.server = nil
	return nil
}

// IsRunning returns whether the server is running
func (m *Manager) IsRunning() bool {
	return m.running
}
