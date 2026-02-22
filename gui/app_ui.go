package main

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// loadCSS loads custom CSS styles for the application
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

// showToast displays a toast notification message
func (m *MauApp) showToast(message string) {
	// Queue toast messages to prevent overflow
	m.toastQueue = append(m.toastQueue, message)

	// If not already processing, start showing toasts
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
