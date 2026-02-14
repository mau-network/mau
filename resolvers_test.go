package mau

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStaticAddress(t *testing.T) {
	t.Run("Returns static address for any fingerprint", func(t *testing.T) {
		expectedAddr := "192.168.1.100:8080"
		resolver := StaticAddress(expectedAddr)

		fpr1, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")
		fpr2, _ := FingerprintFromString("1234567890123456789012345678901234567890")

		addresses := make(chan string, 2)
		ctx := context.Background()

		// Test with first fingerprint
		err := resolver(ctx, fpr1, addresses)
		assert.NoError(t, err)
		addr1 := <-addresses
		assert.Equal(t, expectedAddr, addr1)

		// Test with second fingerprint - should return same address
		err = resolver(ctx, fpr2, addresses)
		assert.NoError(t, err)
		addr2 := <-addresses
		assert.Equal(t, expectedAddr, addr2)
	})

	t.Run("Respects context cancellation", func(t *testing.T) {
		resolver := StaticAddress("192.168.1.100:8080")
		fpr, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")

		// Create cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		addresses := make(chan string, 1)
		err := resolver(ctx, fpr, addresses)
		assert.NoError(t, err)

		// Should not send to channel when context is cancelled
		select {
		case <-addresses:
			t.Error("Should not receive address when context is cancelled")
		case <-time.After(100 * time.Millisecond):
			// Expected: no address sent
		}
	})

	t.Run("Works with different address formats", func(t *testing.T) {
		testCases := []string{
			"localhost:8080",
			"example.com:443",
			"192.168.1.1:9000",
			"[::1]:8080", // IPv6
		}

		fpr, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")

		for _, addr := range testCases {
			t.Run(addr, func(t *testing.T) {
				resolver := StaticAddress(addr)
				addresses := make(chan string, 1)
				err := resolver(context.Background(), fpr, addresses)
				assert.NoError(t, err)
				received := <-addresses
				assert.Equal(t, addr, received)
			})
		}
	})
}

func TestInternetFriendAddress(t *testing.T) {
	t.Run("Returns error when DHT server is nil", func(t *testing.T) {
		account, _ := NewAccount(t.TempDir(), "Test", "test@example.com", "password")
		server, _ := account.Server(nil)
		// Server created but DHT not initialized (nil by default initially)
		server.dhtServer = nil

		resolver := InternetFriendAddress(server)
		fpr, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")
		addresses := make(chan string, 1)

		err := resolver(context.Background(), fpr, addresses)
		assert.ErrorIs(t, err, ErrServerDoesNotAllowLookUp)
	})

	t.Run("Returns address when peer is found via DHT", func(t *testing.T) {
		// Setup bootstrap server
		bootstrapDir := t.TempDir()
		bootstrap, _ := NewAccount(bootstrapDir, "Bootstrap", "bootstrap@example.com", "password")
		bootstrapServer, _ := bootstrap.Server(nil)

		bootstrapListener, bootstrapAddr := TempListener()
		go func() {
			if err := bootstrapServer.Serve(*bootstrapListener, bootstrapAddr); err != nil {
				t.Logf("Bootstrap server error: %v", err)
			}
		}()
		defer bootstrapServer.Close()

		// Wait for DHT server to initialize
		time.Sleep(100 * time.Millisecond)

		// Setup main server
		mainDir := t.TempDir()
		main, _ := NewAccount(mainDir, "Main", "main@example.com", "password")
		mainServer, _ := main.Server([]*Peer{{
			Fingerprint: bootstrap.Fingerprint(),
			Address:     bootstrapAddr,
		}})

		mainListener, mainAddr := TempListener()
		go func() {
			if err := mainServer.Serve(*mainListener, mainAddr); err != nil {
				t.Logf("Main server error: %v", err)
			}
		}()
		defer mainServer.Close()

		// Wait for DHT to initialize and join
		time.Sleep(500 * time.Millisecond)

		// Test lookup
		resolver := InternetFriendAddress(mainServer)
		addresses := make(chan string, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		err := resolver(ctx, bootstrap.Fingerprint(), addresses)
		assert.NoError(t, err)

		select {
		case addr := <-addresses:
			assert.Equal(t, bootstrapAddr, addr)
		case <-time.After(3 * time.Second):
			t.Error("Timeout waiting for address from DHT lookup")
		}
	})

	t.Run("Returns nothing when peer is not found", func(t *testing.T) {
		// Setup server with no bootstrap nodes
		mainDir := t.TempDir()
		main, _ := NewAccount(mainDir, "Main", "main@example.com", "password")
		mainServer, _ := main.Server(nil)

		mainListener, mainAddr := TempListener()
		go func() {
			if err := mainServer.Serve(*mainListener, mainAddr); err != nil {
				t.Logf("Main server error: %v", err)
			}
		}()
		defer mainServer.Close()

		// Wait for DHT to initialize
		time.Sleep(100 * time.Millisecond)

		// Try to find a non-existent peer
		unknownFpr, _ := FingerprintFromString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
		resolver := InternetFriendAddress(mainServer)
		addresses := make(chan string, 1)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := resolver(ctx, unknownFpr, addresses)
		assert.NoError(t, err)

		// Should not receive any address
		select {
		case addr := <-addresses:
			t.Errorf("Should not find unknown peer, but got address: %s", addr)
		case <-time.After(1500 * time.Millisecond):
			// Expected: no address found
		}
	})

	t.Run("Respects context cancellation during lookup", func(t *testing.T) {
		mainDir := t.TempDir()
		main, _ := NewAccount(mainDir, "Main", "main@example.com", "password")
		mainServer, _ := main.Server(nil)

		mainListener, mainAddr := TempListener()
		go func() {
			if err := mainServer.Serve(*mainListener, mainAddr); err != nil {
				t.Logf("Main server error: %v", err)
			}
		}()
		defer mainServer.Close()

		time.Sleep(100 * time.Millisecond)

		fpr, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")
		resolver := InternetFriendAddress(mainServer)
		addresses := make(chan string, 1)

		// Cancel context immediately
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := resolver(ctx, fpr, addresses)
		assert.NoError(t, err)
	})
}

func TestFingerprintResolverConcurrency(t *testing.T) {
	t.Run("Multiple resolvers can be called concurrently", func(t *testing.T) {
		// This tests the pattern used in client.DownloadFriend where multiple resolvers run concurrently
		fpr, _ := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")
		addresses := make(chan string, 3)

		resolver1 := StaticAddress("192.168.1.1:8080")
		resolver2 := StaticAddress("192.168.1.2:8080")
		resolver3 := StaticAddress("192.168.1.3:8080")

		ctx := context.Background()

		// Run all resolvers concurrently (as done in client code)
		go func() {
			_ = resolver1(ctx, fpr, addresses)
		}()
		go func() {
			_ = resolver2(ctx, fpr, addresses)
		}()
		go func() {
			_ = resolver3(ctx, fpr, addresses)
		}()

		// Should receive at least one address
		select {
		case addr := <-addresses:
			assert.NotEmpty(t, addr)
		case <-time.After(1 * time.Second):
			t.Error("Should receive at least one address")
		}
	})
}
