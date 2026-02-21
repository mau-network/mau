// Package main implements the Mau GUI - a GTK4/Adwaita-based client for the Mau P2P social network.
//
// Features:
//   - Post creation with markdown support
//   - Friend management with PGP key verification
//   - P2P server with configurable port
//   - Timeline with pagination and filtering
//   - Automatic retry with exponential backoff
//   - Graceful degradation and offline mode
//
// Architecture:
//   - Modular design with separate view components
//   - Interface-based dependency injection
//   - In-memory LRU cache with TTL
//   - Atomic file operations for data safety
//
// For more information, see README.md
package main

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau"
)

// MauApp is the main application
type MauApp struct {
	app            *adw.Application
	dataDir        string
	configMgr      *ConfigManager
	accountMgr     *AccountManager
	postMgr        *PostManager
	mdRenderer     *MarkdownRenderer
	server         *mau.Server
	serverRunning  bool
	toastOverlay   *adw.ToastOverlay
	mainStack      *adw.ViewStack
	draftSaveTimer glib.SourceHandle
	syncTimer      glib.SourceHandle
	toastQueue     []string
	toastActive    bool
	spinner        *gtk.Spinner

	// Views
	homeView     *HomeView
	friendsView  *FriendsView
	settingsView *SettingsView
}

// NewMauApp creates a new application instance
func NewMauApp(dataDir string) *MauApp {
	app := adw.NewApplication(appID, 0)

	return &MauApp{
		app:     app,
		dataDir: dataDir,
	}
}

// Run starts the application
func (m *MauApp) Run(args []string) int {
	m.app.ConnectActivate(func() {
		if err := m.activate(); err != nil {
			m.showErrorDialog("Initialization Error", err.Error())
		}
	})
	return m.app.Run(args)
}

func (m *MauApp) activate() error {
	// Initialize managers
	m.configMgr = NewConfigManager(m.dataDir)
	m.accountMgr = NewAccountManager(m.dataDir)
	m.mdRenderer = NewMarkdownRenderer()

	// Apply theme
	config := m.configMgr.Get()
	ApplyTheme(m.app, config.DarkMode)

	// Initialize account
	if err := m.accountMgr.Init(); err != nil {
		return fmt.Errorf("account initialization failed: %w", err)
	}

	// Create post manager
	m.postMgr = NewPostManager(m.accountMgr.Account())

	// Update config with account info
	accInfo := m.accountMgr.Info()
	m.configMgr.Update(func(cfg *AppConfig) {
		// Add account if not exists
		exists := false
		for _, acc := range cfg.Accounts {
			if acc.Fingerprint == accInfo.Fingerprint {
				exists = true
				break
			}
		}
		if !exists {
			cfg.Accounts = append(cfg.Accounts, accInfo)
			cfg.LastAccount = accInfo.Fingerprint
		}
	})

	// Create UI
	m.buildUI()

	// Always start server (P2P network - should always be online)
	if err := m.startServer(); err != nil {
		m.showToast(fmt.Sprintf("Warning: Server failed to start: %v", err))
	}

	// Start auto-sync if configured
	if config.AutoSync && config.AutoSyncMinutes > 0 {
		m.startAutoSync()
	}

	return nil
}

func (m *MauApp) buildUI() {
	window := adw.NewApplicationWindow(&m.app.Application)
	window.SetTitle(appTitle)
	window.SetDefaultSize(1000, 800)

	// Toast overlay
	m.toastOverlay = adw.NewToastOverlay()

	// Header bar
	headerBar := adw.NewHeaderBar()
	viewSwitcher := adw.NewViewSwitcher()
	m.mainStack = adw.NewViewStack()
	viewSwitcher.SetStack(m.mainStack)
	headerBar.SetTitleWidget(viewSwitcher)

	// Add loading spinner to header
	m.spinner = gtk.NewSpinner()
	m.spinner.SetVisible(false)
	headerBar.PackEnd(m.spinner)

	// Toolbar view
	toolbarView := adw.NewToolbarView()
	toolbarView.AddTopBar(headerBar)
	toolbarView.SetContent(m.mainStack)

	m.toastOverlay.SetChild(toolbarView)

	// Build views
	m.homeView = NewHomeView(m)
	m.friendsView = NewFriendsView(m)
	m.settingsView = NewSettingsView(m)

	// Add to stack
	m.mainStack.AddTitledWithIcon(m.homeView.Build(), "home", "Home", "user-home-symbolic")
	m.mainStack.AddTitledWithIcon(m.friendsView.Build(), "friends", "Friends", "system-users-symbolic")
	m.mainStack.AddTitledWithIcon(m.settingsView.Build(), "settings", "Settings", "preferences-system-symbolic")

	window.SetContent(m.toastOverlay)

	window.Show()

	// Load CSS after window is shown
	m.loadCSS()
}
