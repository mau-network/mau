package mau

import (
	"net"
	"testing"
)

func TestListenTCP(t *testing.T) {
	// Should work on any system (IPv6-enabled or IPv4-only)
	listener, err := ListenTCP(":0")
	if err != nil {
		t.Fatalf("ListenTCP failed: %v", err)
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	if addr.Port == 0 {
		t.Fatal("Expected non-zero port")
	}

	// Verify we can actually use the listener
	go func() {
		conn, _ := listener.Accept()
		if conn != nil {
			conn.Close()
		}
	}()

	// Try to connect to it
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Could not connect to listener: %v", err)
	}
	conn.Close()
}
