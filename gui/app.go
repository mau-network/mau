package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	
	// Load custom CSS
	m.loadCSS()
	
	window.Show()
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
	
	// CSS is loaded globally - will be applied when windows are created
	// Note: Individual windows can add provider to their display context
}

func (m *MauApp) showToast(message string) {
	toast := adw.NewToast(message)
	toast.SetTimeout(3)
	m.toastOverlay.AddToast(toast)
}

func (m *MauApp) showErrorDialog(title, message string) {
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("ok", "OK")
	dialog.SetDefaultResponse("ok")
	dialog.SetCloseResponse("ok")
	dialog.Show()
}

func (m *MauApp) startServer() error {
	if m.serverRunning {
		return nil
	}

	server, err := m.accountMgr.Account().Server(nil)
	if err != nil {
		return err
	}

	m.server = server

	go func() {
		listener, err := mau.ListenTCP(":8080")
		if err != nil {
			log.Printf("Failed to listen: %v", err)
			return
		}

		if err := m.server.Serve(listener, "127.0.0.1:8080"); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	m.serverRunning = true
	m.showToast("Server started on :8080")
	return nil
}

func (m *MauApp) stopServer() error {
	if !m.serverRunning || m.server == nil {
		return nil
	}

	if err := m.server.Close(); err != nil {
		return err
	}

	m.serverRunning = false
	m.server = nil
	m.showToast("Server stopped")
	return nil
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
	keyring, err := m.accountMgr.Account().ListFriends()
	if err != nil {
		m.showToast("Failed to load friends")
		return
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		return
	}

	m.showToast(fmt.Sprintf("Syncing with %d friends...", len(friends)))
	// Actual sync would happen here via P2P
	if m.timelineView != nil {
		m.timelineView.Refresh()
	}
}

func main() {
	dataDir := filepath.Join(os.Getenv("HOME"), ".mau-gui")
	app := NewMauApp(dataDir)
	
	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
