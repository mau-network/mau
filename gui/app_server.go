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

	server, err := m.createServer()
	if err != nil {
		return err
	}

	m.server = server
	m.launchServerAsync()
	return nil
}

// createServer creates a new server instance
func (m *MauApp) createServer() (*mau.Server, error) {
	server, err := m.accountMgr.Account().Server(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}
	return server, nil
}

// launchServerAsync launches the server in background goroutines
func (m *MauApp) launchServerAsync() {
	config := m.configMgr.Get()
	serverAddr := fmt.Sprintf(":%d", config.ServerPort)
	externalAddr := fmt.Sprintf("127.0.0.1:%d", config.ServerPort)
	startupErr := make(chan error, 1)

	go m.runServerListener(serverAddr, externalAddr, startupErr)
	go m.monitorServerStartup(serverAddr, startupErr)
}

// runServerListener runs the server listener loop
func (m *MauApp) runServerListener(serverAddr, externalAddr string, startupErr chan error) {
	listener, err := mau.ListenTCP(serverAddr)
	if err != nil {
		m.handleListenerError(err, serverAddr, startupErr)
		return
	}

	startupErr <- nil

	if err := m.server.Serve(listener, externalAddr); err != nil {
		m.handleServerError(err)
	}
}

// handleListenerError handles listener creation failures
func (m *MauApp) handleListenerError(err error, serverAddr string, startupErr chan error) {
	log.Printf("Failed to listen on %s: %v", serverAddr, err)
	m.serverRunning = false
	m.server = nil
	startupErr <- err

	glib.IdleAdd(func() bool {
		m.handleServerStartupFailure(err, serverAddr)
		return false
	})
}

// handleServerError handles runtime server errors
func (m *MauApp) handleServerError(err error) {
	log.Printf("Server error: %v", err)
	glib.IdleAdd(func() bool {
		m.showToast(fmt.Sprintf("Server stopped unexpectedly: %v", err))
		m.serverRunning = false
		return false
	})
}

// monitorServerStartup waits for startup completion or timeout
func (m *MauApp) monitorServerStartup(serverAddr string, startupErr chan error) {
	select {
	case err := <-startupErr:
		if err == nil {
			m.notifyServerStarted(serverAddr)
		}
	case <-time.After(serverStartupWait * time.Second):
		m.notifyServerStarted(serverAddr)
	}
}

// notifyServerStarted updates UI after successful startup
func (m *MauApp) notifyServerStarted(serverAddr string) {
	glib.IdleAdd(func() bool {
		m.serverRunning = true
		m.showToast(fmt.Sprintf("Server started on %s", serverAddr))
		return false
	})
}

// handleServerStartupFailure shows error dialog with retry option
func (m *MauApp) handleServerStartupFailure(err error, addr string) {
	friendlyMsg, suggestion := m.parseServerError(err, addr)
	dialog := m.createServerErrorDialog(friendlyMsg, suggestion)
	m.connectServerErrorHandlers(dialog)
	dialog.Show()
}

// parseServerError converts technical errors to user-friendly messages
func (m *MauApp) parseServerError(err error, addr string) (string, string) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "bind") || strings.Contains(errMsg, "address already in use") {
		return errPortInUse, "Try changing the server port in Settings, or stop any other service using the port."
	}
	if strings.Contains(errMsg, "permission denied") {
		return errPermissionDenied, "Try using a port number above 1024, or run with appropriate permissions."
	}
	return fmt.Sprintf("Failed to start server on %s: %v", addr, err), "Check your network configuration and firewall settings."
}

// createServerErrorDialog creates the error dialog UI
func (m *MauApp) createServerErrorDialog(friendlyMsg, suggestion string) *adw.MessageDialog {
	window := m.app.ActiveWindow()
	dialog := adw.NewMessageDialog(window, dialogNetworkError, friendlyMsg+"\n\n"+suggestion)
	dialog.AddResponse("retry", "Retry")
	dialog.AddResponse("offline", "Continue Offline")
	dialog.SetDefaultResponse("retry")
	dialog.SetCloseResponse("offline")
	dialog.SetResponseAppearance("retry", adw.ResponseSuggested)
	return dialog
}

// connectServerErrorHandlers connects dialog response handlers
func (m *MauApp) connectServerErrorHandlers(dialog *adw.MessageDialog) {
	dialog.ConnectResponse(func(responseId string) {
		if responseId == "retry" {
			m.retryServerStart()
		} else {
			m.continueOffline()
		}
	})
}

// retryServerStart retries starting the server
func (m *MauApp) retryServerStart() {
	m.showToast("Retrying server startup...")
	time.AfterFunc(retryDelay*time.Second, func() {
		glib.IdleAdd(func() bool {
			if err := m.startServer(); err != nil {
				m.showToast("Retry failed: " + err.Error())
			}
			return false
		})
	})
}

// continueOffline notifies user about offline mode
func (m *MauApp) continueOffline() {
	m.showToast("Continuing in offline mode. You can start the server later from the Network tab.")
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
