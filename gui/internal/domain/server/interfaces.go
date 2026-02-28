// Package server implements P2P server lifecycle management.
// This package is GTK-agnostic and contains pure server control logic.
package server

// Controller defines the interface for server lifecycle control
type Controller interface {
	// Start starts the P2P server
	Start() error

	// Stop stops the P2P server
	Stop() error

	// IsRunning returns whether the server is currently running
	IsRunning() bool
}

// StatusCallback is called when server status changes
type StatusCallback func(running bool, addr string, err error)

// ErrorParser converts errors to user-friendly messages
type ErrorParser interface {
	// Parse converts a technical error to (friendlyMessage, suggestion)
	Parse(err error, addr string) (string, string)
}
