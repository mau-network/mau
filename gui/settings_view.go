package main

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// SettingsView handles the settings view
type SettingsView struct {
	app              *MauApp
	page             *gtk.Box
	darkModeSwitch   *gtk.Switch
	autoSyncSwitch   *gtk.Switch
	autoSyncInterval *gtk.SpinButton
}

// NewSettingsView creates a new settings view
func NewSettingsView(app *MauApp) *SettingsView {
	return &SettingsView{app: app}
}

// Build creates and returns the view widget
func (sv *SettingsView) Build() *gtk.Box {
	sv.initializePage()
	sv.buildAccountSection()
	sv.buildAppearanceSection()
	sv.buildServerSection()
	sv.buildSyncSection()
	return sv.page
}

func (sv *SettingsView) initializePage() {
	sv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	sv.page.SetMarginTop(12)
	sv.page.SetMarginBottom(12)
	sv.page.SetMarginStart(12)
	sv.page.SetMarginEnd(12)
}

func (sv *SettingsView) buildAccountSection() {
	accountGroup := sv.createAccountGroup()
	sv.addAccountRows(accountGroup)
	sv.page.Append(accountGroup)
}

func (sv *SettingsView) createAccountGroup() *adw.PreferencesGroup {
	accountGroup := adw.NewPreferencesGroup()
	accountGroup.SetTitle("Account")
	accountGroup.SetDescription("Your PGP account information")
	return accountGroup
}

func (sv *SettingsView) addAccountRows(group *adw.PreferencesGroup) {
	group.Add(sv.createNameRow())
	group.Add(sv.createEmailRow())
	group.Add(sv.createFingerprintRow())
}

func (sv *SettingsView) createNameRow() *adw.ActionRow {
	nameRow := adw.NewActionRow()
	nameRow.SetTitle("Name")
	nameRow.SetSubtitle(sv.app.accountMgr.Account().Name())
	return nameRow
}

func (sv *SettingsView) createEmailRow() *adw.ActionRow {
	emailRow := adw.NewActionRow()
	emailRow.SetTitle("Email")
	emailRow.SetSubtitle(sv.app.accountMgr.Account().Email())
	return emailRow
}

func (sv *SettingsView) createFingerprintRow() *adw.ActionRow {
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Fingerprint")
	fpRow.SetSubtitle(sv.app.accountMgr.Account().Fingerprint().String())
	return fpRow
}

func (sv *SettingsView) buildAppearanceSection() {
	appearanceGroup := adw.NewPreferencesGroup()
	appearanceGroup.SetTitle("Appearance")
	appearanceGroup.Add(sv.createDarkModeRow())
	sv.page.Append(appearanceGroup)
}

func (sv *SettingsView) createDarkModeRow() *adw.ActionRow {
	darkModeRow := adw.NewActionRow()
	darkModeRow.SetTitle("Dark Mode")
	darkModeRow.SetSubtitle("Use dark color scheme")
	sv.darkModeSwitch = sv.createDarkModeSwitch()
	darkModeRow.AddSuffix(sv.darkModeSwitch)
	return darkModeRow
}

func (sv *SettingsView) createDarkModeSwitch() *gtk.Switch {
	config := sv.app.configMgr.Get()
	darkSwitch := gtk.NewSwitch()
	darkSwitch.SetActive(config.DarkMode)
	darkSwitch.SetVAlign(gtk.AlignCenter)
	darkSwitch.ConnectStateSet(sv.onDarkModeToggle)
	return darkSwitch
}

func (sv *SettingsView) onDarkModeToggle(state bool) bool {
	sv.app.configMgr.Update(func(cfg *AppConfig) {
		cfg.DarkMode = state
	})
	ApplyTheme(sv.app.app, state)
	status := "enabled"
	if !state {
		status = "disabled"
	}
	sv.app.showToast("Dark mode " + status)
	return false
}

func (sv *SettingsView) buildServerSection() {
	serverGroup := sv.createServerGroup()
	sv.addServerStatusRow(serverGroup)
	sv.addServerPortRow(serverGroup)
	sv.addServerFingerprintRow(serverGroup)
	sv.page.Append(serverGroup)
}

func (sv *SettingsView) createServerGroup() *adw.PreferencesGroup {
	serverGroup := adw.NewPreferencesGroup()
	serverGroup.SetTitle("Network")
	serverGroup.SetDescription("P2P server and network configuration")
	return serverGroup
}

func (sv *SettingsView) addServerStatusRow(group *adw.PreferencesGroup) {
	statusRow := adw.NewActionRow()
	statusRow.SetTitle("Server Status")
	if sv.app.IsRunning() {
		config := sv.app.configMgr.Get()
		statusRow.SetSubtitle("Running on port " + fmt.Sprintf("%d", config.ServerPort))
	} else {
		statusRow.SetSubtitle("Not running")
	}
	group.Add(statusRow)
}

func (sv *SettingsView) addServerPortRow(group *adw.PreferencesGroup) {
	config := sv.app.configMgr.Get()
	portRow := adw.NewActionRow()
	portRow.SetTitle("Server Port")
	portRow.SetSubtitle("Port for P2P server (requires restart)")
	portSpin := sv.createPortSpinButton(config.ServerPort)
	portRow.AddSuffix(portSpin)
	group.Add(portRow)
}

func (sv *SettingsView) createPortSpinButton(currentPort int) *gtk.SpinButton {
	portSpin := gtk.NewSpinButtonWithRange(1024, 65535, 1)
	portSpin.SetValue(float64(currentPort))
	portSpin.SetVAlign(gtk.AlignCenter)
	portSpin.ConnectValueChanged(func() {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.ServerPort = int(portSpin.Value())
		})
		sv.app.showToast("Server port updated (restart app to apply)")
	})
	return portSpin
}

func (sv *SettingsView) addServerFingerprintRow(group *adw.PreferencesGroup) {
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Network Fingerprint")
	fpRow.SetSubtitle(sv.app.accountMgr.Account().Fingerprint().String())
	group.Add(fpRow)
}

func (sv *SettingsView) buildSyncSection() {
	syncGroup := adw.NewPreferencesGroup()
	syncGroup.SetTitle("Synchronization")
	config := sv.app.configMgr.Get()
	syncGroup.Add(sv.createAutoSyncRow(config))
	syncGroup.Add(sv.createIntervalRow(config))
	sv.page.Append(syncGroup)
}

func (sv *SettingsView) createAutoSyncRow(config AppConfig) *adw.ActionRow {
	autoSyncRow := adw.NewActionRow()
	autoSyncRow.SetTitle("Auto-sync")
	autoSyncRow.SetSubtitle("Automatically sync with friends")
	sv.autoSyncSwitch = sv.createAutoSyncSwitch(config)
	autoSyncRow.AddSuffix(sv.autoSyncSwitch)
	return autoSyncRow
}

func (sv *SettingsView) createAutoSyncSwitch(config AppConfig) *gtk.Switch {
	syncSwitch := gtk.NewSwitch()
	syncSwitch.SetActive(config.AutoSync)
	syncSwitch.SetVAlign(gtk.AlignCenter)
	syncSwitch.ConnectStateSet(sv.onAutoSyncToggle)
	return syncSwitch
}

func (sv *SettingsView) onAutoSyncToggle(state bool) bool {
	sv.app.configMgr.Update(func(cfg *AppConfig) {
		cfg.AutoSync = state
	})
	if state {
		sv.app.startAutoSync()
	}
	status := "enabled"
	if !state {
		status = "disabled"
	}
	sv.app.showToast("Auto-sync " + status)
	return false
}

func (sv *SettingsView) createIntervalRow(config AppConfig) *adw.ActionRow {
	intervalRow := adw.NewActionRow()
	intervalRow.SetTitle("Sync Interval")
	intervalRow.SetSubtitle("Minutes between automatic syncs")
	sv.autoSyncInterval = sv.createIntervalSpinButton(config)
	intervalRow.AddSuffix(sv.autoSyncInterval)
	return intervalRow
}

func (sv *SettingsView) createIntervalSpinButton(config AppConfig) *gtk.SpinButton {
	intervalSpin := gtk.NewSpinButtonWithRange(5, 1440, 5)
	intervalSpin.SetValue(float64(config.AutoSyncMinutes))
	intervalSpin.SetVAlign(gtk.AlignCenter)
	intervalSpin.ConnectValueChanged(func() {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.AutoSyncMinutes = int(sv.autoSyncInterval.Value())
		})
	})
	return intervalSpin
}
