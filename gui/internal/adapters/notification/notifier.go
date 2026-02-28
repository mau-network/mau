// Package notification provides toast and dialog notification adapters for GTK.
package notification

import (
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
)

const (
	toastTimeout     = 3      // seconds
	toastDisplayTime = 1      // seconds between sequential toasts
)

// Notifier provides toast and dialog notifications
type Notifier struct {
	overlay     *adw.ToastOverlay
	app         *adw.Application
	toastQueue  []string
	toastActive bool
}

// NewNotifier creates a notification manager
func NewNotifier(overlay *adw.ToastOverlay, app *adw.Application) *Notifier {
	return &Notifier{
		overlay:     overlay,
		app:         app,
		toastQueue:  make([]string, 0),
		toastActive: false,
	}
}

// ShowToast displays a toast notification message
func (n *Notifier) ShowToast(message string) {
	// Queue toast messages to prevent overflow
	n.toastQueue = append(n.toastQueue, message)

	// If not already processing, start showing toasts
	if !n.toastActive {
		n.processToastQueue()
	}
}

// processToastQueue shows queued toast messages sequentially
func (n *Notifier) processToastQueue() {
	if len(n.toastQueue) == 0 {
		n.toastActive = false
		return
	}

	n.toastActive = true

	// Get next toast
	message := n.toastQueue[0]
	n.toastQueue = n.toastQueue[1:]

	// Show toast
	toast := adw.NewToast(message)
	toast.SetTimeout(toastTimeout)
	n.overlay.AddToast(toast)

	// Process next toast after delay
	glib.TimeoutSecondsAdd(toastDisplayTime, func() bool {
		n.processToastQueue()
		return false
	})
}

// ShowError displays an error dialog
func (n *Notifier) ShowError(title, message string) {
	window := n.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, title, message)
	dialog.AddResponse("ok", "OK")
	dialog.SetDefaultResponse("ok")
	dialog.SetCloseResponse("ok")
	dialog.Show()
}

// ShowConfirm displays a confirmation dialog
func (n *Notifier) ShowConfirm(title, message string, onConfirm func()) {
	window := n.app.ActiveWindow()
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
