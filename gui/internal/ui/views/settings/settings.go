// Package settings provides the settings view for configuring the application.
package settings

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/mau-network/mau-gui-poc/internal/adapters/notification"
	"github.com/mau-network/mau-gui-poc/internal/domain/account"
	"github.com/mau-network/mau-gui-poc/internal/domain/config"
	"github.com/mau-network/mau-gui-poc/internal/domain/server"
	"github.com/mau-network/mau-gui-poc/internal/ui/theme"
)

// View handles the settings view
type View struct {
	configMgr  *config.Manager
	accountMgr *account.Manager
	serverMgr  *server.Manager
	notifier   *notification.Notifier
	gtkApp     *adw.Application

	// Callbacks
	onAutoSyncToggled func(enabled bool)

	// UI components
	page             *gtk.Box
	darkModeSwitch   *gtk.Switch
	autoSyncSwitch   *gtk.Switch
	autoSyncInterval *gtk.SpinButton
}

// Config holds view configuration
type Config struct {
	ConfigMgr         *config.Manager
	AccountMgr        *account.Manager
	ServerMgr         *server.Manager
	Notifier          *notification.Notifier
	GtkApp            *adw.Application
	OnAutoSyncToggled func(enabled bool)
}

// New creates a new settings view
func New(cfg Config) *View {
	v := &View{
		configMgr:         cfg.ConfigMgr,
		accountMgr:        cfg.AccountMgr,
		serverMgr:         cfg.ServerMgr,
		notifier:          cfg.Notifier,
		gtkApp:            cfg.GtkApp,
		onAutoSyncToggled: cfg.OnAutoSyncToggled,
	}
	v.buildUI()
	return v
}

// Widget returns the view's widget
func (v *View) Widget() *gtk.Box {
	return v.page
}

// buildUI creates and returns the view widget
func (v *View) buildUI() {
	v.page = gtk.NewBox(gtk.OrientationVertical, 12)
	v.page.SetMarginTop(12)
	v.page.SetMarginBottom(12)
	v.page.SetMarginStart(12)
	v.page.SetMarginEnd(12)

	// Account info
	v.buildAccountSection()

	// Appearance
	v.buildAppearanceSection()

	// Server settings
	v.buildServerSection()

	// Sync settings
	v.buildSyncSection()
}

func (v *View) buildAccountSection() {
	accountGroup := adw.NewPreferencesGroup()
	accountGroup.SetTitle("Account")
	accountGroup.SetDescription("Your PGP account information")

	acc := v.accountMgr.Account()

	// Name (read-only)
	nameRow := adw.NewActionRow()
	nameRow.SetTitle("Name")
	nameRow.SetSubtitle(acc.Name())
	accountGroup.Add(nameRow)

	// Email (read-only)
	emailRow := adw.NewActionRow()
	emailRow.SetTitle("Email")
	emailRow.SetSubtitle(acc.Email())
	accountGroup.Add(emailRow)

	// Fingerprint (read-only)
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Fingerprint")
	fpRow.SetSubtitle(acc.Fingerprint().String())
	accountGroup.Add(fpRow)

	v.page.Append(accountGroup)
}

func (v *View) buildAppearanceSection() {
	appearanceGroup := adw.NewPreferencesGroup()
	appearanceGroup.SetTitle("Appearance")

	darkModeRow := adw.NewActionRow()
	darkModeRow.SetTitle("Dark Mode")
	darkModeRow.SetSubtitle("Use dark color scheme")

	cfg := v.configMgr.Get()
	v.darkModeSwitch = gtk.NewSwitch()
	v.darkModeSwitch.SetActive(cfg.DarkMode)
	v.darkModeSwitch.SetVAlign(gtk.AlignCenter)
	v.darkModeSwitch.ConnectStateSet(func(state bool) bool {
		v.configMgr.Update(func(cfg *config.AppConfig) {
			cfg.DarkMode = state
		})
		theme.Apply(v.gtkApp, state)
		v.notifier.ShowToast("Dark mode " + map[bool]string{true: "enabled", false: "disabled"}[state])
		return false
	})
	darkModeRow.AddSuffix(v.darkModeSwitch)
	appearanceGroup.Add(darkModeRow)

	v.page.Append(appearanceGroup)
}

func (v *View) buildServerSection() {
	serverGroup := adw.NewPreferencesGroup()
	serverGroup.SetTitle("Network")
	serverGroup.SetDescription("P2P server and network configuration")

	// Server status
	statusRow := adw.NewActionRow()
	statusRow.SetTitle("Server Status")
	if v.serverMgr.IsRunning() {
		cfg := v.configMgr.Get()
		statusRow.SetSubtitle("Running on port " + fmt.Sprintf("%d", cfg.ServerPort))
	} else {
		statusRow.SetSubtitle("Not running")
	}
	serverGroup.Add(statusRow)

	// Server port configuration
	cfg := v.configMgr.Get()
	portRow := adw.NewActionRow()
	portRow.SetTitle("Server Port")
	portRow.SetSubtitle("Port for P2P server (requires restart)")

	portSpin := gtk.NewSpinButtonWithRange(1024, 65535, 1)
	portSpin.SetValue(float64(cfg.ServerPort))
	portSpin.SetVAlign(gtk.AlignCenter)
	portSpin.ConnectValueChanged(func() {
		v.configMgr.Update(func(cfg *config.AppConfig) {
			cfg.ServerPort = int(portSpin.Value())
		})
		v.notifier.ShowToast("Server port updated (restart app to apply)")
	})
	portRow.AddSuffix(portSpin)
	serverGroup.Add(portRow)

	// Network Fingerprint
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Network Fingerprint")
	fpRow.SetSubtitle(v.accountMgr.Account().Fingerprint().String())
	serverGroup.Add(fpRow)

	v.page.Append(serverGroup)
}

func (v *View) buildSyncSection() {
	syncGroup := adw.NewPreferencesGroup()
	syncGroup.SetTitle("Synchronization")

	cfg := v.configMgr.Get()

	autoSyncRow := adw.NewActionRow()
	autoSyncRow.SetTitle("Auto-sync")
	autoSyncRow.SetSubtitle("Automatically sync with friends")

	v.autoSyncSwitch = gtk.NewSwitch()
	v.autoSyncSwitch.SetActive(cfg.AutoSync)
	v.autoSyncSwitch.SetVAlign(gtk.AlignCenter)
	v.autoSyncSwitch.ConnectStateSet(func(state bool) bool {
		v.configMgr.Update(func(cfg *config.AppConfig) {
			cfg.AutoSync = state
		})
		// Notify parent app via callback
		if v.onAutoSyncToggled != nil {
			v.onAutoSyncToggled(state)
		}
		v.notifier.ShowToast("Auto-sync " + map[bool]string{true: "enabled", false: "disabled"}[state])
		return false
	})
	autoSyncRow.AddSuffix(v.autoSyncSwitch)
	syncGroup.Add(autoSyncRow)

	intervalRow := adw.NewActionRow()
	intervalRow.SetTitle("Sync Interval")
	intervalRow.SetSubtitle("Minutes between automatic syncs")

	v.autoSyncInterval = gtk.NewSpinButtonWithRange(5, 1440, 5)
	v.autoSyncInterval.SetValue(float64(cfg.AutoSyncMinutes))
	v.autoSyncInterval.SetVAlign(gtk.AlignCenter)
	v.autoSyncInterval.ConnectValueChanged(func() {
		v.configMgr.Update(func(cfg *config.AppConfig) {
			cfg.AutoSyncMinutes = int(v.autoSyncInterval.Value())
		})
	})
	intervalRow.AddSuffix(v.autoSyncInterval)
	syncGroup.Add(intervalRow)

	v.page.Append(syncGroup)
}
