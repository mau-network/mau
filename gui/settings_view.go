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
	sv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	sv.page.SetMarginTop(12)
	sv.page.SetMarginBottom(12)
	sv.page.SetMarginStart(12)
	sv.page.SetMarginEnd(12)

	// Account info
	sv.buildAccountSection()

	// Appearance
	sv.buildAppearanceSection()

	// Server settings
	sv.buildServerSection()

	// Sync settings
	sv.buildSyncSection()

	return sv.page
}

func (sv *SettingsView) buildAccountSection() {
	accountGroup := adw.NewPreferencesGroup()
	accountGroup.SetTitle("Account")
	accountGroup.SetDescription("Your PGP account and identities")

	// Fingerprint (read-only)
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Fingerprint")
	fpRow.SetSubtitle(sv.app.accountMgr.Account().Fingerprint().String())
	accountGroup.Add(fpRow)

	sv.page.Append(accountGroup)
	
	// Identities section
	sv.buildIdentitiesSection()
}

func (sv *SettingsView) buildIdentitiesSection() {
	identitiesGroup := adw.NewPreferencesGroup()
	identitiesGroup.SetTitle("Identities")
	identitiesGroup.SetDescription("PGP keys can have multiple name/email identities")

	identities := sv.app.accountMgr.Account().ListIdentities()
	
	for _, identityStr := range identities {
		row := adw.NewActionRow()
		row.SetTitle(identityStr)
		
		isPrimary := sv.app.accountMgr.Account().IsPrimaryIdentity(identityStr)
		
		if isPrimary {
			row.SetSubtitle("Primary identity")
			icon := gtk.NewImageFromIconName("emblem-default-symbolic")
			row.AddPrefix(icon)
		} else {
			// Add "Set Primary" button for non-primary identities
			setPrimaryBtn := gtk.NewButton()
			setPrimaryBtn.SetLabel("Set Primary")
			setPrimaryBtn.SetVAlign(gtk.AlignCenter)
			identStr := identityStr // Capture for closure
			setPrimaryBtn.ConnectClicked(func() {
				sv.setPrimaryIdentity(identStr)
			})
			row.AddSuffix(setPrimaryBtn)
		}
		
		identitiesGroup.Add(row)
	}

	// Add new identity button
	addIdentityRow := adw.NewActionRow()
	addIdentityRow.SetTitle("Add New Identity")
	addIdentityRow.SetSubtitle("Add another name/email to this account")
	
	addBtn := gtk.NewButton()
	addBtn.SetLabel("Add Identity")
	addBtn.SetVAlign(gtk.AlignCenter)
	addBtn.ConnectClicked(func() {
		sv.showAddIdentityDialog()
	})
	addIdentityRow.AddSuffix(addBtn)
	identitiesGroup.Add(addIdentityRow)

	sv.page.Append(identitiesGroup)
}

func (sv *SettingsView) setPrimaryIdentity(identityName string) {
	// Check for cached passphrase first
	cachedPass, hasCached := sv.app.accountMgr.GetCachedPassphrase()
	
	if hasCached {
		// Try with cached passphrase
		err := sv.app.accountMgr.Account().SetPrimaryIdentity(identityName, cachedPass)
		if err == nil {
			sv.app.showToast("Primary identity updated!")
			sv.refreshAccountSection()
			return
		}
		// If failed, fall through to prompt (might be wrong cached passphrase)
	}

	// Prompt for passphrase
	window := sv.app.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, "Set Primary Identity", "")
	dialog.SetBody("Enter your account passphrase to change primary identity")
	
	contentBox := gtk.NewBox(gtk.OrientationVertical, 12)
	contentBox.SetMarginTop(12)
	contentBox.SetMarginBottom(12)
	contentBox.SetMarginStart(12)
	contentBox.SetMarginEnd(12)

	passphraseLabel := gtk.NewLabel("Account Passphrase:")
	passphraseLabel.SetXAlign(0)
	passphraseEntry := gtk.NewPasswordEntry()
	passphraseEntry.SetShowPeekIcon(true)
	contentBox.Append(passphraseLabel)
	contentBox.Append(passphraseEntry)

	dialog.SetExtraChild(contentBox)
	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("set", "Set Primary")
	dialog.SetDefaultResponse("set")
	dialog.SetCloseResponse("cancel")
	dialog.SetResponseAppearance("set", adw.ResponseSuggested)

	dialog.ConnectResponse(func(response string) {
		if response == "set" {
			passphrase := passphraseEntry.Text()
			if passphrase == "" {
				sv.app.ShowError("Passphrase Required", "You must enter your account passphrase")
				return
			}

			err := sv.app.accountMgr.Account().SetPrimaryIdentity(identityName, passphrase)
			if err != nil {
				sv.app.ShowError("Failed to Set Primary", err.Error())
				return
			}

			// Cache passphrase on success
			sv.app.accountMgr.CachePassphrase(passphrase)

			sv.app.showToast("Primary identity updated!")
			sv.refreshAccountSection()
		}
	})

	dialog.Show()
}

func (sv *SettingsView) refreshAccountSection() {
	// Rebuild the entire page
	// Remove all current content
	for {
		child := sv.page.FirstChild()
		if child == nil {
			break
		}
		sv.page.Remove(child)
	}

	// Rebuild all sections
	sv.buildAccountSection()
	sv.buildAppearanceSection()
	sv.buildServerSection()
	sv.buildSyncSection()
}

func (sv *SettingsView) showAddIdentityDialog() {
	window := sv.app.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, "Add New Identity", "")
	
	// Content area with form
	contentBox := gtk.NewBox(gtk.OrientationVertical, 12)
	contentBox.SetMarginTop(12)
	contentBox.SetMarginBottom(12)
	contentBox.SetMarginStart(12)
	contentBox.SetMarginEnd(12)

	// Name entry
	nameLabel := gtk.NewLabel("Name:")
	nameLabel.SetXAlign(0)
	nameEntry := gtk.NewEntry()
	nameEntry.SetPlaceholderText("Your Name")
	contentBox.Append(nameLabel)
	contentBox.Append(nameEntry)

	// Email entry
	emailLabel := gtk.NewLabel("Email:")
	emailLabel.SetXAlign(0)
	emailEntry := gtk.NewEntry()
	emailEntry.SetPlaceholderText("your@email.com")
	contentBox.Append(emailLabel)
	contentBox.Append(emailEntry)

	// Passphrase entry
	passphraseLabel := gtk.NewLabel("Account Passphrase:")
	passphraseLabel.SetXAlign(0)
	passphraseEntry := gtk.NewPasswordEntry()
	passphraseEntry.SetShowPeekIcon(true)
	contentBox.Append(passphraseLabel)
	contentBox.Append(passphraseEntry)

	dialog.SetExtraChild(contentBox)
	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("add", "Add Identity")
	dialog.SetDefaultResponse("add")
	dialog.SetCloseResponse("cancel")
	dialog.SetResponseAppearance("add", adw.ResponseSuggested)

	dialog.ConnectResponse(func(response string) {
		if response == "add" {
			sv.addIdentity(nameEntry.Text(), emailEntry.Text(), passphraseEntry.Text())
		}
	})

	dialog.Show()
}

func (sv *SettingsView) addIdentity(name, email, passphrase string) {
	if name == "" || email == "" {
		sv.app.ShowError("Invalid Input", "Name and email cannot be empty")
		return
	}

	if passphrase == "" {
		sv.app.ShowError("Passphrase Required", "You must enter your account passphrase to add a new identity")
		return
	}

	err := sv.app.accountMgr.Account().AddIdentity(name, email, passphrase)
	if err != nil {
		sv.app.ShowError("Failed to Add Identity", err.Error())
		return
	}

	// Cache passphrase on success for future operations
	sv.app.accountMgr.CachePassphrase(passphrase)

	sv.app.showToast("Identity added successfully!")
	sv.refreshAccountSection()
}

func (sv *SettingsView) buildAppearanceSection() {
	appearanceGroup := adw.NewPreferencesGroup()
	appearanceGroup.SetTitle("Appearance")

	darkModeRow := adw.NewActionRow()
	darkModeRow.SetTitle("Dark Mode")
	darkModeRow.SetSubtitle("Use dark color scheme")

	config := sv.app.configMgr.Get()
	sv.darkModeSwitch = gtk.NewSwitch()
	sv.darkModeSwitch.SetActive(config.DarkMode)
	sv.darkModeSwitch.SetVAlign(gtk.AlignCenter)
	sv.darkModeSwitch.ConnectStateSet(func(state bool) bool {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.DarkMode = state
		})
		ApplyTheme(sv.app.app, state)
		sv.app.showToast("Dark mode " + map[bool]string{true: "enabled", false: "disabled"}[state])
		return false
	})
	darkModeRow.AddSuffix(sv.darkModeSwitch)
	appearanceGroup.Add(darkModeRow)

	sv.page.Append(appearanceGroup)
}

func (sv *SettingsView) buildServerSection() {
	serverGroup := adw.NewPreferencesGroup()
	serverGroup.SetTitle("Network")
	serverGroup.SetDescription("P2P server and network configuration")

	// Server status (always running)
	statusRow := adw.NewActionRow()
	statusRow.SetTitle("Server Status")
	if sv.app.IsRunning() {
		config := sv.app.configMgr.Get()
		statusRow.SetSubtitle("Running on port " + fmt.Sprintf("%d", config.ServerPort))
	} else {
		statusRow.SetSubtitle("Not running")
	}
	serverGroup.Add(statusRow)

	// Server port configuration
	config := sv.app.configMgr.Get()
	portRow := adw.NewActionRow()
	portRow.SetTitle("Server Port")
	portRow.SetSubtitle("Port for P2P server (requires restart)")

	portSpin := gtk.NewSpinButtonWithRange(1024, 65535, 1)
	portSpin.SetValue(float64(config.ServerPort))
	portSpin.SetVAlign(gtk.AlignCenter)
	portSpin.ConnectValueChanged(func() {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.ServerPort = int(portSpin.Value())
		})
		sv.app.showToast("Server port updated (restart app to apply)")
	})
	portRow.AddSuffix(portSpin)
	serverGroup.Add(portRow)

	// Fingerprint
	fpRow := adw.NewActionRow()
	fpRow.SetTitle("Network Fingerprint")
	fpRow.SetSubtitle(sv.app.accountMgr.Account().Fingerprint().String())
	serverGroup.Add(fpRow)

	sv.page.Append(serverGroup)
}

func (sv *SettingsView) buildSyncSection() {
	syncGroup := adw.NewPreferencesGroup()
	syncGroup.SetTitle("Synchronization")

	config := sv.app.configMgr.Get()

	autoSyncRow := adw.NewActionRow()
	autoSyncRow.SetTitle("Auto-sync")
	autoSyncRow.SetSubtitle("Automatically sync with friends")

	sv.autoSyncSwitch = gtk.NewSwitch()
	sv.autoSyncSwitch.SetActive(config.AutoSync)
	sv.autoSyncSwitch.SetVAlign(gtk.AlignCenter)
	sv.autoSyncSwitch.ConnectStateSet(func(state bool) bool {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.AutoSync = state
		})
		if state {
			sv.app.startAutoSync()
		}
		sv.app.showToast("Auto-sync " + map[bool]string{true: "enabled", false: "disabled"}[state])
		return false
	})
	autoSyncRow.AddSuffix(sv.autoSyncSwitch)
	syncGroup.Add(autoSyncRow)

	intervalRow := adw.NewActionRow()
	intervalRow.SetTitle("Sync Interval")
	intervalRow.SetSubtitle("Minutes between automatic syncs")

	sv.autoSyncInterval = gtk.NewSpinButtonWithRange(5, 1440, 5)
	sv.autoSyncInterval.SetValue(float64(config.AutoSyncMinutes))
	sv.autoSyncInterval.SetVAlign(gtk.AlignCenter)
	sv.autoSyncInterval.ConnectValueChanged(func() {
		sv.app.configMgr.Update(func(cfg *AppConfig) {
			cfg.AutoSyncMinutes = int(sv.autoSyncInterval.Value())
		})
	})
	intervalRow.AddSuffix(sv.autoSyncInterval)
	syncGroup.Add(intervalRow)

	sv.page.Append(syncGroup)
}
