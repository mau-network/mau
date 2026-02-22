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
	app             *adw.Application
	dataDir         string
	configMgr       *ConfigManager
	accountMgr      *AccountManager
	postMgr         *PostManager
	mdRenderer      *MarkdownRenderer
	server          *mau.Server
	serverRunning   bool
	toastOverlay    *adw.ToastOverlay
	mainStack       *adw.ViewStack
	draftSaveTimer  glib.SourceHandle
	syncTimer       glib.SourceHandle
	toastQueue      []string
	toastActive     bool
	spinner         *gtk.Spinner
	statusIndicator *gtk.Label

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
	m.initializeManagers()
	m.applyThemeFromConfig()
	if err := m.initializeAccount(); err != nil {
		return err
	}
	m.createPostManager()
	m.updateConfigWithAccount()
	m.buildUI()
	m.startServerIfNeeded()
	m.startAutoSyncIfConfigured()
	return nil
}

func (m *MauApp) initializeManagers() {
	m.configMgr = NewConfigManager(m.dataDir)
	m.accountMgr = NewAccountManager(m.dataDir)
	m.mdRenderer = NewMarkdownRenderer()
}

func (m *MauApp) applyThemeFromConfig() {
	config := m.configMgr.Get()
	ApplyTheme(m.app, config.DarkMode)
}

func (m *MauApp) initializeAccount() error {
	if err := m.accountMgr.Init(); err != nil {
		return fmt.Errorf("account initialization failed: %w", err)
	}
	return nil
}

func (m *MauApp) createPostManager() {
	m.postMgr = NewPostManager(m.accountMgr.Account())
}

func (m *MauApp) updateConfigWithAccount() {
	accInfo := m.accountMgr.Info()
	m.configMgr.Update(func(cfg *AppConfig) {
		m.addAccountIfNotExists(cfg, accInfo)
	})
}

func (m *MauApp) addAccountIfNotExists(cfg *AppConfig, accInfo AccountInfo) {
	for _, acc := range cfg.Accounts {
		if acc.Fingerprint == accInfo.Fingerprint {
			return
		}
	}
	cfg.Accounts = append(cfg.Accounts, accInfo)
	cfg.LastAccount = accInfo.Fingerprint
}

func (m *MauApp) startServerIfNeeded() {
	if err := m.startServer(); err != nil {
		m.showToast(fmt.Sprintf("Warning: Server failed to start: %v", err))
	}
}

func (m *MauApp) startAutoSyncIfConfigured() {
	config := m.configMgr.Get()
	if config.AutoSync && config.AutoSyncMinutes > 0 {
		m.startAutoSync()
	}
}

func (m *MauApp) buildUI() {
	window := m.createMainWindow()
	m.setupToastOverlay()
	headerBar := m.buildHeaderBar()
	toolbarView := m.buildToolbarView(headerBar)
	m.toastOverlay.SetChild(toolbarView)
	m.buildAndAddViews()
	window.SetContent(m.toastOverlay)
	window.Show()
	m.loadCSS()
	m.updateNetworkStatus()
}

func (m *MauApp) createMainWindow() *adw.ApplicationWindow {
	window := adw.NewApplicationWindow(&m.app.Application)
	window.SetTitle(appTitle)
	window.SetDefaultSize(1000, 800)
	return window
}

func (m *MauApp) setupToastOverlay() {
	m.toastOverlay = adw.NewToastOverlay()
}

func (m *MauApp) buildHeaderBar() *adw.HeaderBar {
	headerBar := adw.NewHeaderBar()
	viewSwitcher := adw.NewViewSwitcher()
	m.mainStack = adw.NewViewStack()
	viewSwitcher.SetStack(m.mainStack)
	headerBar.SetTitleWidget(viewSwitcher)
	m.addHeaderBarWidgets(headerBar)
	return headerBar
}

func (m *MauApp) addHeaderBarWidgets(headerBar *adw.HeaderBar) {
	m.addSpinnerToHeader(headerBar)
	m.addStatusIndicator(headerBar)
}

func (m *MauApp) addSpinnerToHeader(headerBar *adw.HeaderBar) {
	m.spinner = gtk.NewSpinner()
	m.spinner.SetVisible(false)
	headerBar.PackEnd(m.spinner)
}

func (m *MauApp) addStatusIndicator(headerBar *adw.HeaderBar) {
	statusBox := gtk.NewBox(gtk.OrientationHorizontal, 4)
	m.statusIndicator = gtk.NewLabel("🔴 Offline")
	m.statusIndicator.AddCSSClass("status-label")
	statusBox.Append(m.statusIndicator)
	headerBar.PackEnd(statusBox)
}

func (m *MauApp) buildToolbarView(headerBar *adw.HeaderBar) *adw.ToolbarView {
	toolbarView := adw.NewToolbarView()
	toolbarView.AddTopBar(headerBar)
	toolbarView.SetContent(m.mainStack)
	return toolbarView
}

func (m *MauApp) buildAndAddViews() {
	m.homeView = NewHomeView(m)
	m.friendsView = NewFriendsView(m)
	m.settingsView = NewSettingsView(m)
	m.addViewsToStack()
}

func (m *MauApp) addViewsToStack() {
	m.mainStack.AddTitledWithIcon(m.homeView.Build(), "home", "Home", "user-home-symbolic")
	m.mainStack.AddTitledWithIcon(m.friendsView.Build(), "friends", "Friends", "system-users-symbolic")
	m.mainStack.AddTitledWithIcon(m.settingsView.Build(), "settings", "Settings", "preferences-system-symbolic")
}
