package main

import (
	"fmt"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// NetworkView handles the network/server view
type NetworkView struct {
	app          *MauApp
	page         *gtk.Box
	serverToggle *gtk.Switch
	serverStatus *gtk.Label
}

// NewNetworkView creates a new network view
func NewNetworkView(app *MauApp) *NetworkView {
	return &NetworkView{app: app}
}

// Build creates and returns the view widget
func (nv *NetworkView) Build() *gtk.Box {
	nv.page = gtk.NewBox(gtk.OrientationVertical, 12)
	nv.page.SetMarginTop(12)
	nv.page.SetMarginBottom(12)
	nv.page.SetMarginStart(12)
	nv.page.SetMarginEnd(12)

	// Server control
	serverGroup := adw.NewPreferencesGroup()
	serverGroup.SetTitle("P2P Server")
	serverGroup.SetDescription("Control local P2P server")

	serverRow := adw.NewActionRow()
	serverRow.SetTitle("Server Status")

	nv.serverStatus = gtk.NewLabel("Stopped")
	nv.serverStatus.AddCSSClass("status-indicator")
	nv.serverStatus.AddCSSClass("status-stopped")
	serverRow.AddSuffix(nv.serverStatus)

	nv.serverToggle = gtk.NewSwitch()
	nv.serverToggle.SetActive(false)
	nv.serverToggle.ConnectStateSet(func(state bool) bool {
		if state {
			if err := nv.app.Start(); err != nil {
				nv.app.ShowError("Server Error", err.Error())
				nv.serverToggle.SetActive(false)
				return false
			}
			nv.UpdateStatus(true)
		} else {
			if err := nv.app.Stop(); err != nil {
				nv.app.ShowError("Server Error", err.Error())
				return false
			}
			nv.UpdateStatus(false)
		}
		return false
	})
	serverRow.AddSuffix(nv.serverToggle)

	serverGroup.Add(serverRow)
	nv.page.Append(serverGroup)

	// Network info
	infoGroup := adw.NewPreferencesGroup()
	infoGroup.SetTitle("Network Information")

	fprRow := adw.NewActionRow()
	fprRow.SetTitle("Your Fingerprint")
	fprRow.SetSubtitle(nv.app.accountMgr.Account().Fingerprint().String())
	infoGroup.Add(fprRow)

	// Server port info
	config := nv.app.configMgr.Get()
	portRow := adw.NewActionRow()
	portRow.SetTitle("Server Port")
	portRow.SetSubtitle(fmt.Sprintf("%d (configurable in settings)", config.ServerPort))
	infoGroup.Add(portRow)

	nv.page.Append(infoGroup)

	// Update initial status
	nv.UpdateStatus(nv.app.IsRunning())

	return nv.page
}

// UpdateStatus updates the server status indicator
func (nv *NetworkView) UpdateStatus(running bool) {
	config := nv.app.configMgr.Get()
	if running {
		nv.serverStatus.SetText(fmt.Sprintf("Running on :%d", config.ServerPort))
		nv.serverStatus.RemoveCSSClass("status-stopped")
		nv.serverStatus.AddCSSClass("status-running")
	} else {
		nv.serverStatus.SetText("Stopped")
		nv.serverStatus.RemoveCSSClass("status-running")
		nv.serverStatus.AddCSSClass("status-stopped")
	}
}
