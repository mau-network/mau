// Package app provides application orchestration and lifecycle management.
package app

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/config"
	"github.com/mau-network/mau-gui-poc/internal/domain/post"
	"github.com/mau-network/mau-gui-poc/internal/domain/server"
	"github.com/mau-network/mau-gui-poc/internal/ui/theme"
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

	// UI state
	window       *adw.ApplicationWindow
	toastOverlay *adw.ToastOverlay
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

	// Build UI (placeholder for now - will be filled in migration)
	a.buildPlaceholderUI()

	// Start server
	if err := a.serverMgr.Start(cfg.ServerPort); err != nil {
		a.notifier.ShowToast(fmt.Sprintf("Warning: Server failed to start: %v", err))
	}

	a.window.Show()
}

// buildPlaceholderUI creates a temporary UI (to be replaced with full implementation)
func (a *App) buildPlaceholderUI() {
	label := adw.NewStatusPage()
	label.SetTitle("Mau GUI - Architecture Migration")
	label.SetDescription("UI views being migrated to new architecture.\nSee gui/README.md for details.")

	a.toastOverlay.SetChild(label)
	a.window.SetContent(a.toastOverlay)
}
