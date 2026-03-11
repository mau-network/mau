// Package app provides application orchestration and lifecycle management.
package app

import (
	"fmt"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/config"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
	"github.com/mau-network/mau-gui-poc/internal/domain/server"
	"github.com/mau-network/mau-gui-poc/internal/ui/theme"
	"github.com/mau-network/mau-gui-poc/internal/ui/views/friends"
	"github.com/mau-network/mau-gui-poc/internal/ui/views/home"
	"github.com/mau-network/mau-gui-poc/internal/ui/views/settings"
	"github.com/mau-network/mau-gui-poc/internal/ui/views/timeline"
	"github.com/mau-network/mau-gui-poc/pkg/retry"
)

// App coordinates the application lifecycle
type App struct {
	gtkApp *adw.Application

	// Domain managers
	configMgr  *config.Manager
	accountMgr *account.Manager
	postMgr    *post.Manager
	serverMgr  *server.Manager

	// Adapters
	notifier *notification.Notifier

	// UI components
	window       *adw.ApplicationWindow
	toastOverlay *adw.ToastOverlay
	mainStack    *adw.ViewStack

	// Views
	homeView     *home.View
	friendsView  *friends.View
	settingsView *settings.View
	timelineView *timeline.View

	// State
	syncTimer glib.SourceHandle
}

// Config holds application configuration
type Config struct {
	ConfigMgr  *config.Manager
	AccountMgr *account.Manager
	PostMgr    *post.Manager
	ServerMgr  *server.Manager
}

const appID = "com.mau.gui"

// New creates a new application instance
func New(cfg Config) *App {
	gtkApp := adw.NewApplication(appID, 0)

	app := &App{
		gtkApp:     gtkApp,
		configMgr:  cfg.ConfigMgr,
		accountMgr: cfg.AccountMgr,
		postMgr:    cfg.PostMgr,
		serverMgr:  cfg.ServerMgr,
	}

	gtkApp.ConnectActivate(app.activate)
	return app
}

// Run starts the application
func (a *App) Run(args []string) int {
	return a.gtkApp.Run(args)
}

// activate is called when the application is activated
func (a *App) activate() {
	// Apply theme
	cfg := a.configMgr.Get()
	theme.Apply(a.gtkApp, cfg.DarkMode)

	// Create main window
	a.window = adw.NewApplicationWindow(&a.gtkApp.Application)
	a.window.SetTitle("Mau")
	a.window.SetDefaultSize(1000, 800)

	// Create toast overlay
	a.toastOverlay = adw.NewToastOverlay()

	// Initialize notifier
	a.notifier = notification.NewNotifier(a.toastOverlay, a.gtkApp)

	// Build UI with real views
	a.buildUI()

	// Start server
	if err := a.serverMgr.Start(cfg.ServerPort); err != nil {
		a.notifier.ShowToast(fmt.Sprintf("Warning: Server failed to start: %v", err))
	}

	// Start auto-sync if enabled
	if cfg.AutoSync {
		a.startAutoSync()
	}

	a.window.Show()
}

// buildUI creates the main UI with navigation and views
func (a *App) buildUI() {
	// Create main container
	mainBox := gtk.NewBox(gtk.OrientationVertical, 0)

	// Create header bar with view switcher
	headerBar := adw.NewHeaderBar()
	
	viewSwitcher := adw.NewViewSwitcher()
	a.mainStack = adw.NewViewStack()
	viewSwitcher.SetStack(a.mainStack)
	headerBar.SetTitleWidget(viewSwitcher)

	mainBox.Append(headerBar)

	// Create views
	a.homeView = home.New(home.Config{
		AccountMgr: a.accountMgr,
		PostMgr:    a.postMgr,
		Notifier:   a.notifier,
	})

	a.friendsView = friends.New(friends.Config{
		AccountMgr: a.accountMgr,
		Notifier:   a.notifier,
		GtkApp:     a.gtkApp,
	})

	a.timelineView = timeline.New(timeline.Config{
		AccountMgr:      a.accountMgr,
		PostMgr:         a.postMgr,
		Notifier:        a.notifier,
		OnSyncRequested: func() { a.syncFriends() },
	})

	a.settingsView = settings.New(settings.Config{
		ConfigMgr:         a.configMgr,
		AccountMgr:        a.accountMgr,
		ServerMgr:         a.serverMgr,
		Notifier:          a.notifier,
		GtkApp:            a.gtkApp,
		OnAutoSyncToggled: func(enabled bool) { a.handleAutoSyncToggle(enabled) },
	})

	// Add views to stack
	a.mainStack.AddTitledWithIcon(a.homeView.Widget(), "home", "Home", "user-home-symbolic")
	a.mainStack.AddTitledWithIcon(a.timelineView.Widget(), "timeline", "Timeline", "emblem-synchronizing-symbolic")
	a.mainStack.AddTitledWithIcon(a.friendsView.Widget(), "friends", "Friends", "system-users-symbolic")
	a.mainStack.AddTitledWithIcon(a.settingsView.Widget(), "settings", "Settings", "preferences-system-symbolic")

	// Wrap stack in scrolled window
	scrolled := gtk.NewScrolledWindow()
	scrolled.SetChild(a.mainStack)
	scrolled.SetVExpand(true)

	mainBox.Append(scrolled)

	// Set up toast overlay
	a.toastOverlay.SetChild(mainBox)
	a.window.SetContent(a.toastOverlay)
}

// handleAutoSyncToggle starts or stops auto-sync
func (a *App) handleAutoSyncToggle(enabled bool) {
	if enabled {
		a.startAutoSync()
	} else {
		if a.syncTimer != 0 {
			glib.SourceRemove(a.syncTimer)
			a.syncTimer = 0
		}
	}
}

// startAutoSync starts automatic synchronization with friends
func (a *App) startAutoSync() {
	cfg := a.configMgr.Get()
	if cfg.AutoSyncMinutes <= 0 {
		return
	}

	// Clear existing timer
	if a.syncTimer != 0 {
		glib.SourceRemove(a.syncTimer)
	}

	interval := uint(cfg.AutoSyncMinutes * 60 * 1000)
	a.syncTimer = glib.TimeoutAdd(interval, func() bool {
		currentCfg := a.configMgr.Get()
		if currentCfg.AutoSync {
			a.syncFriends()
			return true
		}
		return false
	})
}

// syncFriends synchronizes posts with friends
func (a *App) syncFriends() {
	// Use retry logic for sync operation
	cfg := retry.RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 2 * time.Second,
		MaxDelay:     10 * time.Second,
		Multiplier:   2.0,
	}

	err := retry.RetryWithContext(cfg, func(attempt int, err error) {
		// Show retry notification
		a.notifier.ShowToast(fmt.Sprintf("Sync failed (attempt %d/3), retrying...", attempt))
	}, func() error {
		return a.performSync()
	})

	if err != nil {
		a.notifier.ShowToast("Sync failed: " + err.Error())
		return
	}

	a.notifier.ShowToast("Sync complete")
}

// performSync performs the actual synchronization
func (a *App) performSync() error {
	keyring, err := a.accountMgr.Account().ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends: %w", err)
	}

	friends := keyring.FriendsSet()
	if len(friends) == 0 {
		// Not an error, just no friends
		a.notifier.ShowToast("No friends to sync")
		return nil
	}

	a.notifier.ShowToast(fmt.Sprintf("Syncing with friends... (%d friends)", len(friends)))

	// Actual sync would happen here via P2P
	// For now, just refresh the views
	if a.homeView != nil {
		a.homeView.Refresh()
	}
	if a.timelineView != nil {
		a.timelineView.Refresh()
	}

	return nil
}
