package main

import (
	"strings"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// loadCSS loads custom CSS styles for the application
func (m *MauApp) loadCSS() {
	css := m.buildCSSString()
	provider := m.createCSSProvider(css)
	m.applyCSSToDisplay(provider)
}

func (m *MauApp) buildCSSString() string {
	parts := []string{
		m.cssPostStyles(),
		m.cssTagStyles(),
		m.cssUIStyles(),
		m.cssStatusStyles(),
	}
	return strings.Join(parts, "\n")
}

func (m *MauApp) cssPostStyles() string {
	return strings.Join([]string{
		`.post-card {`,
		`  padding: 12px;`,
		`  margin: 6px;`,
		`}`,
		`.post-header {`,
		`  font-weight: bold;`,
		`  margin-bottom: 6px;`,
		`}`,
		`.post-body {`,
		`  margin: 6px 0;`,
		`}`,
		`.post-footer {`,
		`  font-size: 0.9em;`,
		`  opacity: 0.7;`,
		`  margin-top: 6px;`,
		`}`,
	}, "\n")
}

func (m *MauApp) cssTagStyles() string {
	return `.tag-label {
		background: alpha(currentColor, 0.1);
		border-radius: 4px;
		padding: 2px 8px;
		margin: 2px;
		font-size: 0.85em;
	}
	
	.char-counter {
		font-size: 0.9em;
		opacity: 0.7;
	}`
}

func (m *MauApp) cssStatusStyles() string {
	return `.status-indicator {
		font-weight: bold;
	}
	
	.status-running {
		color: @success_color;
	}
	
	.status-stopped {
		color: @error_color;
	}`
}

func (m *MauApp) cssUIStyles() string {
	return `.preview-box {
		background: alpha(currentColor, 0.05);
		border-radius: 8px;
		padding: 12px;
		margin: 6px 0;
	}`
}

func (m *MauApp) createCSSProvider(css string) *gtk.CSSProvider {
	provider := gtk.NewCSSProvider()
	provider.LoadFromData(css)
	return provider
}

func (m *MauApp) applyCSSToDisplay(provider *gtk.CSSProvider) {
	display := gdk.DisplayGetDefault()
	if display != nil {
		gtk.StyleContextAddProviderForDisplay(
			display,
			provider,
			gtk.STYLE_PROVIDER_PRIORITY_APPLICATION,
		)
	}
}

// showToast displays a toast notification message
func (m *MauApp) showToast(message string) {
	m.toastQueue = append(m.toastQueue, message)
	if !m.toastActive {
		m.processToastQueue()
	}
}

// processToastQueue shows queued toast messages sequentially
func (m *MauApp) processToastQueue() {
	if len(m.toastQueue) == 0 {
		m.toastActive = false
		return
	}
	m.toastActive = true
	message := m.toastQueue[0]
	m.toastQueue = m.toastQueue[1:]
	m.displayToast(message)
	m.scheduleNextToast()
}

func (m *MauApp) displayToast(message string) {
	toast := adw.NewToast(message)
	toast.SetTimeout(toastTimeout)
	m.toastOverlay.AddToast(toast)
}

func (m *MauApp) scheduleNextToast() {
	glib.TimeoutSecondsAdd(toastDisplayTime, func() bool {
		m.processToastQueue()
		return false
	})
}

// showErrorDialog displays an error dialog
func (m *MauApp) showErrorDialog(title, message string) {
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("ok", "OK")
	dialog.SetDefaultResponse("ok")
	dialog.SetCloseResponse("ok")
	dialog.Show()
}

// showConfirmDialog displays a confirmation dialog
func (m *MauApp) showConfirmDialog(title, message string, onConfirm func()) {
	window := m.app.ActiveWindow()
	dialog := m.createConfirmDialog(window, title, message)
	m.connectConfirmResponse(dialog, onConfirm)
	dialog.Show()
}

func (m *MauApp) createConfirmDialog(window *gtk.Window, title, message string) *adw.MessageDialog {
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("cancel", "Cancel")
	dialog.AddResponse("confirm", "Confirm")
	dialog.SetDefaultResponse("cancel")
	dialog.SetCloseResponse("cancel")
	dialog.SetResponseAppearance("confirm", adw.ResponseDestructive)
	return dialog
}

func (m *MauApp) connectConfirmResponse(dialog *adw.MessageDialog, onConfirm func()) {
	dialog.ConnectResponse(func(response string) {
		if response == "confirm" && onConfirm != nil {
			onConfirm()
		}
	})
}

// setLoading shows or hides the loading spinner
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

// ShowToast implements ToastNotifier interface
func (m *MauApp) ShowToast(message string) {
	m.showToast(message)
}

// ShowError implements ToastNotifier interface
func (m *MauApp) ShowError(title, message string) {
	m.showErrorDialog(title, message)
}
