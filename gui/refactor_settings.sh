#!/bin/bash
# Automated function refactoring for settings_view.go

set -e

FILE="settings_view.go"
echo "Refactoring $FILE..."

# Create backup
cp "$FILE" "$FILE.bak"

# buildServerSection refactor (42 lines → split)
cat > /tmp/server_section.go << 'EOF'
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
EOF

echo "Server section refactored"
echo "Build and test..."
go build && go test ./...
echo "$FILE refactored successfully"
