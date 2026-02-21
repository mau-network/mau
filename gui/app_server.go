package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/mau-network/mau"
)

// startServer starts the P2P server
func (m *MauApp) startServer() error {
	if m.serverRunning {
		return nil
	}

	server, err := m.accountMgr.Account().Server(nil)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	m.server = server

	// Get configured port
	config := m.configMgr.Get()
	serverAddr := fmt.Sprintf(":%d", config.ServerPort)
	externalAddr := fmt.Sprintf("127.0.0.1:%d", config.ServerPort)

	// Channel to capture startup errors
	startupErr := make(chan error, 1)

	go func() {
		listener, err := mau.ListenTCP(serverAddr)
		if err != nil {
			log.Printf("Failed to listen on %s: %v", serverAddr, err)
			m.serverRunning = false
			m.server = nil
			startupErr <- err

			// Show error dialog with retry option (using glib.IdleAdd for GTK thread safety)
			glib.IdleAdd(func() bool {
				m.handleServerStartupFailure(err, serverAddr)
				return false
			})
			return
		}

		// Signal successful startup
		startupErr <- nil

		if err := m.server.Serve(listener, externalAddr); err != nil {
			log.Printf("Server error on %s: %v", serverAddr, err)

			// Show error if server fails during operation
			glib.IdleAdd(func() bool {
				m.showToast(fmt.Sprintf("Server stopped unexpectedly: %v", err))
				m.serverRunning = false
				return false
			})
		}
	}()

	// Wait briefly for startup result
	go func() {
		select {
		case err := <-startupErr:
			if err == nil {
				// Success
				glib.IdleAdd(func() bool {
					m.serverRunning = true
					m.showToast(fmt.Sprintf("Server started on %s", serverAddr))
					return false
				})
			}
			// Error case handled in goroutine above
		case <-time.After(serverStartupWait * time.Second):
			// Assume success if no error within 2 seconds
			glib.IdleAdd(func() bool {
				m.serverRunning = true
				m.showToast(fmt.Sprintf("Server started on %s", serverAddr))
				return false
			})
		}
	}()

	return nil
}

// handleServerStartupFailure handles server startup errors with user-friendly dialogs
func (m *MauApp) handleServerStartupFailure(err error, addr string) {
	// Determine user-friendly error message
	errMsg := err.Error()
	var friendlyMsg string
	var suggestion string

	if strings.Contains(errMsg, "bind") || strings.Contains(errMsg, "address already in use") {
		friendlyMsg = errPortInUse
		suggestion = "Try changing the server port in Settings, or stop any other service using the port."
	} else if strings.Contains(errMsg, "permission denied") {
		friendlyMsg = errPermissionDenied
		suggestion = "Try using a port number above 1024, or run with appropriate permissions."
	} else {
		friendlyMsg = fmt.Sprintf("Failed to start server on %s: %v", addr, err)
		suggestion = "Check your network configuration and firewall settings."
	}

	// Show error dialog with details and retry option
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(
		window,
		dialogNetworkError,
		friendlyMsg+"\n\n"+suggestion,
	)
	dialog.AddResponse("retry", "Retry")
	dialog.AddResponse("offline", "Continue Offline")
	dialog.SetDefaultResponse("retry")
	dialog.SetCloseResponse("offline")
	dialog.SetResponseAppearance("retry", adw.ResponseSuggested)

	dialog.ConnectResponse(func(responseId string) {
		if responseId == "retry" {
			// Retry starting the server
			m.showToast("Retrying server startup...")
			time.AfterFunc(retryDelay*time.Second, func() {
				glib.IdleAdd(func() bool {
					if err := m.startServer(); err != nil {
						m.showToast("Retry failed: " + err.Error())
					}
					return false
				})
			})
		} else {
			// Continue in offline mode
			m.showToast("Continuing in offline mode. You can start the server later from the Network tab.")
		}
	})

	dialog.Show()
}

// stopServer stops the P2P server
func (m *MauApp) stopServer() error {
	if !m.serverRunning || m.server == nil {
		return nil
	}

	if err := m.server.Close(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	m.serverRunning = false
	m.server = nil
	m.showToast("Server stopped")
	return nil
}

// Start implements ServerController interface
func (m *MauApp) Start() error {
	return m.startServer()
}

// Stop implements ServerController interface
func (m *MauApp) Stop() error {
	return m.stopServer()
}

// IsRunning implements ServerController interface
func (m *MauApp) IsRunning() bool {
	return m.serverRunning
}
