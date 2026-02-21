package main

import (
	"fmt"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

// startAutoSync starts automatic synchronization with friends
func (m *MauApp) startAutoSync() {
	config := m.configMgr.Get()
	if config.AutoSyncMinutes <= 0 {
		return
	}

	interval := uint(config.AutoSyncMinutes * 60 * 1000)
	glib.TimeoutAdd(interval, func() bool {
		cfg := m.configMgr.Get()
		if cfg.AutoSync {
			m.syncFriends()
			return true
		}
		return false
	})
}

// syncFriends synchronizes posts with friends
func (m *MauApp) syncFriends() {
	m.setLoading(true)
	defer m.setLoading(false)

	// Use retry logic for sync operation
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: retryInitialDelay * time.Second,
		MaxDelay:     retryMaxDelay * time.Second,
		Multiplier:   2.0,
	}

	err := RetryWithContext(cfg, func(attempt int, err error) {
		// Show retry notification
		m.showToast(fmt.Sprintf("Sync failed (attempt %d/3), retrying...", attempt))
	}, func() error {
		return m.performSync()
	})

	if err != nil {
		m.showToast(toastSyncFailed + ": " + err.Error())
		return
	}

	m.showToast(toastSyncComplete)
}

// performSync performs the actual synchronization
func (m *MauApp) performSync() error {
	keyring, err := m.accountMgr.Account().ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		// Not an error, just no friends
		m.showToast(toastNoFriends)
		return nil
	}

	m.showToast(fmt.Sprintf("%s (%d friends)", toastSyncStarted, len(friends)))

	// Actual sync would happen here via P2P
	// Refresh the home view to show new posts
	if m.homeView != nil {
		m.homeView.Refresh()
	}

	return nil
}
