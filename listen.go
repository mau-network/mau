package mau

import (
	"net"
)

// ListenTCP creates a TCP listener that works on both IPv6-enabled and IPv4-only systems.
// It tries to bind to both IPv4 and IPv6 (dual-stack) first, then falls back to IPv4 only.
func ListenTCP(addr string) (net.Listener, error) {
	// Try dual-stack (IPv6 + IPv4) first
	listener, err := net.Listen("tcp", addr)
	if err == nil {
		return listener, nil
	}

	// If dual-stack fails (e.g., IPv6 disabled), fall back to IPv4 only
	listener, err = net.Listen("tcp4", addr)
	if err != nil {
		return nil, err
	}

	return listener, nil
}
