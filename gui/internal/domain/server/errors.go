package server

import (
	"fmt"
	"strings"
)

// ErrorMessages contains user-friendly error messages
const (
	ErrPortInUse         = "Port is already in use"
	ErrPermissionDenied  = "Permission denied to bind port"
)

// DefaultErrorParser provides user-friendly error parsing
type DefaultErrorParser struct{}

// Parse converts technical errors to user-friendly messages
func (p *DefaultErrorParser) Parse(err error, addr string) (string, string) {
	errMsg := err.Error()

	if strings.Contains(errMsg, "bind") || strings.Contains(errMsg, "address already in use") {
		return ErrPortInUse, "Try changing the server port in Settings, or stop any other service using the port."
	}
	if strings.Contains(errMsg, "permission denied") {
		return ErrPermissionDenied, "Try using a port number above 1024, or run with appropriate permissions."
	}
	return fmt.Sprintf("Failed to start server on %s: %v", addr, err),
		"Check your network configuration and firewall settings."
}
