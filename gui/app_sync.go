package main

import (
	"fmt"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/mau-network/mau"
)

// startAutoSync starts automatic synchronization with friends
func (m *MauApp) startAutoSync() {
	config := m.configMgr.Get()
	if config.AutoSyncMinutes <= 0 {
		return
	}
	interval := uint(config.AutoSyncMinutes * 60 * 1000)
	glib.TimeoutAdd(interval, m.autoSyncCallback)
}

func (m *MauApp) autoSyncCallback() bool {
	cfg := m.configMgr.Get()
	if cfg.AutoSync {
		m.syncFriends()
		return true
	}
	return false
}

// syncFriends synchronizes posts with friends
func (m *MauApp) syncFriends() {
	m.setLoading(true)
	defer m.setLoading(false)
	cfg := m.createRetryConfig()
	err := RetryWithContext(cfg, m.onSyncRetry, m.performSync)
	m.handleSyncResult(err)
}

func (m *MauApp) createRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: retryInitialDelay * time.Second,
		MaxDelay:     retryMaxDelay * time.Second,
		Multiplier:   2.0,
	}
}

func (m *MauApp) onSyncRetry(attempt int, err error) {
	m.showToast(fmt.Sprintf("Sync failed (attempt %d/3), retrying...", attempt))
}

func (m *MauApp) handleSyncResult(err error) {
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
		m.showToast(toastNoFriends)
		return nil
	}
	m.performActualSync(friends)
	return nil
}

func (m *MauApp) performActualSync(friends []*mau.Friend) {
	m.showToast(fmt.Sprintf("%s (%d friends)", toastSyncStarted, len(friends)))
	if m.homeView != nil {
		m.homeView.Refresh()
	}
}
