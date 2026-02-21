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
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
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
	timelineView *TimelineView
	friendsView  *FriendsView
	networkView  *NetworkView
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

	// Auto-start server if configured
	if config.AutoStartServer {
		m.startServer()
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
	m.timelineView = NewTimelineView(m)
	m.friendsView = NewFriendsView(m)
	m.networkView = NewNetworkView(m)
	m.settingsView = NewSettingsView(m)

	// Add to stack
	m.mainStack.AddTitledWithIcon(m.homeView.Build(), "home", "Home", "user-home-symbolic")
	m.mainStack.AddTitledWithIcon(m.timelineView.Build(), "timeline", "Timeline", "view-list-symbolic")
	m.mainStack.AddTitledWithIcon(m.friendsView.Build(), "friends", "Friends", "system-users-symbolic")
	m.mainStack.AddTitledWithIcon(m.networkView.Build(), "network", "Network", "network-workgroup-symbolic")
	m.mainStack.AddTitledWithIcon(m.settingsView.Build(), "settings", "Settings", "preferences-system-symbolic")

	window.SetContent(m.toastOverlay)

	window.Show()

	// Load CSS after window is shown
	m.loadCSS()
}

func (m *MauApp) loadCSS() {
	css := `
		.post-card {
			padding: 12px;
			margin: 6px;
		}
		
		.post-header {
			font-weight: bold;
			margin-bottom: 6px;
		}
		
		.post-body {
			margin: 6px 0;
		}
		
		.post-footer {
			font-size: 0.9em;
			opacity: 0.7;
			margin-top: 6px;
		}
		
		.tag-label {
			background: alpha(currentColor, 0.1);
			border-radius: 4px;
			padding: 2px 8px;
			margin: 2px;
			font-size: 0.85em;
		}
		
		.char-counter {
			font-size: 0.9em;
			opacity: 0.7;
		}
		
		.status-indicator {
			font-weight: bold;
		}
		
		.status-running {
			color: @success_color;
		}
		
		.status-stopped {
			color: @error_color;
		}
		
		.preview-box {
			background: alpha(currentColor, 0.05);
			border-radius: 8px;
			padding: 12px;
			margin: 6px 0;
		}
	`

	provider := gtk.NewCSSProvider()
	provider.LoadFromData(css)

	// Apply CSS to the default display (works for all windows)
	display := gdk.DisplayGetDefault()
	if display != nil {
		gtk.StyleContextAddProviderForDisplay(
			display,
			provider,
			gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
		)
	}
}

func (m *MauApp) showToast(message string) {
	// Queue toast messages to prevent overflow
	m.toastQueue = append(m.toastQueue, message)

	// If not already processing, start showing toasts
	if !m.toastActive {
		m.processToastQueue()
	}
}

func (m *MauApp) processToastQueue() {
	if len(m.toastQueue) == 0 {
		m.toastActive = false
		return
	}

	m.toastActive = true

	// Get next toast
	message := m.toastQueue[0]
	m.toastQueue = m.toastQueue[1:]

	// Show toast
	toast := adw.NewToast(message)
	toast.SetTimeout(toastTimeout)
	m.toastOverlay.AddToast(toast)

	// Process next toast after delay
	glib.TimeoutSecondsAdd(toastDisplayTime, func() bool {
		m.processToastQueue()
		return false
	})
}

func (m *MauApp) showErrorDialog(title, message string) {
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("ok", "OK")
	dialog.SetDefaultResponse("ok")
	dialog.SetCloseResponse("ok")
	dialog.Show()
}

func (m *MauApp) showConfirmDialog(title, message string, onConfirm func()) {
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("confirm", "Confirm")
	dialog.SetDefaultResponse("cancel")
	dialog.SetCloseResponse("cancel")
	dialog.SetResponseAppearance("confirm", adw.ResponseDestructive)

	dialog.ConnectResponse(func(response string) {
		if response == "confirm" && onConfirm != nil {
			onConfirm()
		}
	})

	dialog.Show()
}

func (m *MauApp) startServer() error {
	if m.serverRunning {
		return nil
	}

	server, err := m.accountMgr.Account().Server(nil)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	m.server = server

	// Get configured port
	config := m.configMgr.Get()
	serverAddr := fmt.Sprintf(":%d", config.ServerPort)
	externalAddr := fmt.Sprintf("127.0.0.1:%d", config.ServerPort)

	// Channel to capture startup errors
	startupErr := make(chan error, 1)

	go func() {
		listener, err := mau.ListenTCP(serverAddr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", serverAddr, err)
			m.serverRunning = false
			m.server = nil
			startupErr <- err

			// Show error dialog with retry option (using glib.IdleAdd for GTK thread safety)
			glib.IdleAdd(func() bool {
				m.handleServerStartupFailure(err, serverAddr)
				return false
			})
			return
		}

		// Signal successful startup
		startupErr <- nil

		if err := m.server.Serve(listener, externalAddr); err != nil {
			log.Printf("Server error on %s: %v", serverAddr, err)

			// Show error if server fails during operation
			glib.IdleAdd(func() bool {
				m.showToast(fmt.Sprintf("Server stopped unexpectedly: %v", err))
				m.serverRunning = false
				return false
			})
		}
	}()

	// Wait briefly for startup result
	go func() {
		select {
		case err := <-startupErr:
			if err == nil {
				// Success
				glib.IdleAdd(func() bool {
					m.serverRunning = true
					m.showToast(fmt.Sprintf("Server started on %s", serverAddr))
					return false
				})
			}
			// Error case handled in goroutine above
		case <-time.After(serverStartupWait * time.Second):
			// Assume success if no error within 2 seconds
			glib.IdleAdd(func() bool {
				m.serverRunning = true
				m.showToast(fmt.Sprintf("Server started on %s", serverAddr))
				return false
			})
		}
	}()

	return nil
}

func (m *MauApp) handleServerStartupFailure(err error, addr string) {
	// Determine user-friendly error message
	errMsg := err.Error()
	var friendlyMsg string
	var suggestion string

	if strings.Contains(errMsg, "bind") || strings.Contains(errMsg, "address already in use") {
		friendlyMsg = errPortInUse
		suggestion = "Try changing the server port in Settings, or stop any other service using the port."
	} else if strings.Contains(errMsg, "permission denied") {
		friendlyMsg = errPermissionDenied
		suggestion = "Try using a port number above 1024, or run with appropriate permissions."
	} else {
		friendlyMsg = fmt.Sprintf("Failed to start server on %s: %v", addr, err)
		suggestion = "Check your network configuration and firewall settings."
	}

	// Show error dialog with details and retry option
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(
		window,
		dialogNetworkError,
		friendlyMsg+"\n\n"+suggestion,
	)
	dialog.AddResponse("retry", "Retry")
	dialog.AddResponse("offline", "Continue Offline")
	dialog.SetDefaultResponse("retry")
	dialog.SetCloseResponse("offline")
	dialog.SetResponseAppearance("retry", adw.ResponseSuggested)

	dialog.ConnectResponse(func(responseId string) {
		if responseId == "retry" {
			// Retry starting the server
			m.showToast("Retrying server startup...")
			time.AfterFunc(retryDelay*time.Second, func() {
				glib.IdleAdd(func() bool {
					if err := m.startServer(); err != nil {
						m.showToast("Retry failed: " + err.Error())
					}
					return false
				})
			})
		} else {
			// Continue in offline mode
			m.showToast("Continuing in offline mode. You can start the server later from the Network tab.")
		}
	})

	dialog.Show()
}

func (m *MauApp) stopServer() error {
	if !m.serverRunning || m.server == nil {
		return nil
	}

	if err := m.server.Close(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	m.serverRunning = false
	m.server = nil
	m.showToast("Server stopped")
	return nil
}

// Interface implementations for ServerController
func (m *MauApp) Start() error {
	return m.startServer()
}

func (m *MauApp) Stop() error {
	return m.stopServer()
}

func (m *MauApp) IsRunning() bool {
	return m.serverRunning
}

// Interface implementations for ToastNotifier
func (m *MauApp) ShowToast(message string) {
	m.showToast(message)
}

func (m *MauApp) ShowError(title, message string) {
	m.showErrorDialog(title, message)
}

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
	// For now, just refresh the timeline
	if m.timelineView != nil {
		m.timelineView.Refresh()
	}

	return nil
}

func (m *MauApp) setLoading(loading bool) {
	if m.spinner != nil {
		if loading {
			m.spinner.Start()
			m.spinner.SetVisible(true)
		} else {
			m.spinner.Stop()
			m.spinner.SetVisible(false)
		}
	}
}

func main() {
	// Allow data dir to be overridden via environment variable for testing
	dataDir := os.Getenv("MAU_GUI_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mau-gui")
	}

	app := NewMauApp(dataDir)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
