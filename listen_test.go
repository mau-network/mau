package mau

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListenTCP(t *testing.T) {
	t.Run("succeeds with :0 (random port)", func(t *testing.T) {
		listener, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		addr := listener.Addr().String()
		assert.NotEmpty(t, addr)
		assert.Contains(t, addr, ":")
	})

	t.Run("succeeds with localhost:0", func(t *testing.T) {
		listener, err := ListenTCP("localhost:0")
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		addr := listener.Addr().String()
		assert.NotEmpty(t, addr)
	})

	t.Run("succeeds with 127.0.0.1:0", func(t *testing.T) {
		listener, err := ListenTCP("127.0.0.1:0")
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		addr := listener.Addr().String()
		assert.NotEmpty(t, addr)
		assert.Contains(t, addr, "127.0.0.1:")
	})

	t.Run("fails with invalid address", func(t *testing.T) {
		listener, err := ListenTCP("invalid:address:format")
		assert.Error(t, err)
		assert.Nil(t, listener)
	})

	t.Run("returns valid net.Listener interface", func(t *testing.T) {
		listener, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		// Verify it implements net.Listener
		var _ net.Listener = listener

		// Verify we can get the address
		addr := listener.Addr()
		assert.NotNil(t, addr)
		assert.Equal(t, "tcp", addr.Network())
	})

	t.Run("can accept connections", func(t *testing.T) {
		listener, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener)
		defer listener.Close()

		addr := listener.Addr().String()

		// Connect to the listener in a goroutine
		connected := make(chan bool)
		go func() {
			conn, err := net.Dial("tcp", addr)
			if err == nil {
				conn.Close()
				connected <- true
			} else {
				connected <- false
			}
		}()

		// Accept the connection
		conn, err := listener.Accept()
		require.NoError(t, err)
		require.NotNil(t, conn)
		conn.Close()

		// Verify the client connected successfully
		assert.True(t, <-connected)
	})

	t.Run("multiple listeners on different ports", func(t *testing.T) {
		listener1, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener1)
		defer listener1.Close()

		listener2, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener2)
		defer listener2.Close()

		addr1 := listener1.Addr().String()
		addr2 := listener2.Addr().String()

		// Addresses should be different (different ports)
		assert.NotEqual(t, addr1, addr2)
	})

	t.Run("port already in use returns error", func(t *testing.T) {
		// First listener
		listener1, err := ListenTCP(":0")
		require.NoError(t, err)
		require.NotNil(t, listener1)
		defer listener1.Close()

		// Get the port that was assigned
		_, port, err := net.SplitHostPort(listener1.Addr().String())
		require.NoError(t, err)

		// Try to listen on the same port
		listener2, err := ListenTCP(":" + port)
		assert.Error(t, err)
		assert.Nil(t, listener2)
	})
}
