package tests

import (
	"context"
	"testing"
	"time"

	"github.com/mau-network/mau/e2e/internal/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSinglePeerHealthCheck tests that a single peer can start and respond to health checks
func TestSinglePeerHealthCheck(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Create test environment
	env, err := testenv.NewEnvironment("test-single-peer")
	require.NoError(t, err, "Failed to create test environment")

	// Ensure cleanup
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = env.Stop(cleanupCtx)
	}()

	// Start environment with 1 peer
	err = env.Start(ctx, 1)
	require.NoError(t, err, "Failed to start environment")

	// Verify we have exactly 1 peer
	assert.Len(t, env.Peers, 1, "Expected exactly 1 peer")

	peer := env.Peers[0]
	assert.NotEmpty(t, peer.ContainerID, "Peer should have container ID")
	assert.NotEmpty(t, peer.Name, "Peer should have name")
	assert.Greater(t, peer.HTTPPort, 0, "Peer should have HTTP port assigned")
	assert.True(t, peer.Running, "Peer should be running")

	// TODO: Add actual health check HTTP request when API is implemented
	// For now, we verify the container started successfully (waitingFor strategy succeeded)

	t.Logf("✓ Peer %s started successfully", peer.Name)
	t.Logf("  Container: %s", peer.ContainerID[:12])
	t.Logf("  HTTP Port: %d", peer.HTTPPort)
}

// TestTwoPeerDiscovery tests that two peers can discover each other via DHT
func TestTwoPeerDiscovery(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	ctx := context.Background()

	// Create test environment
	env, err := testenv.NewEnvironment("test-two-peer-discovery")
	require.NoError(t, err, "Failed to create test environment")

	// Ensure cleanup
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = env.Stop(cleanupCtx)
	}()

	// Start environment with 2 peers
	err = env.Start(ctx, 2)
	require.NoError(t, err, "Failed to start environment")

	// Verify we have exactly 2 peers
	assert.Len(t, env.Peers, 2, "Expected exactly 2 peers")

	peer0 := env.Peers[0]
	peer1 := env.Peers[1]

	// Verify both peers are running
	assert.True(t, peer0.Running, "Peer 0 should be running")
	assert.True(t, peer1.Running, "Peer 1 should be running")

	// Verify peers have different container IDs
	assert.NotEqual(t, peer0.ContainerID, peer1.ContainerID,
		"Peers should have different container IDs")

	// Verify peers have different HTTP ports
	assert.NotEqual(t, peer0.HTTPPort, peer1.HTTPPort,
		"Peers should have different HTTP ports")

	// Both peers should be on the same Docker network
	assert.NotEmpty(t, env.NetworkName, "Environment should have a network")

	t.Logf("✓ Two peers started successfully")
	t.Logf("  Peer 0: %s (HTTP: %d)", peer0.ContainerID[:12], peer0.HTTPPort)
	t.Logf("  Peer 1: %s (HTTP: %d)", peer1.ContainerID[:12], peer1.HTTPPort)
	t.Logf("  Network: %s", env.NetworkName)

	// TODO: Implement DHT discovery test
	// This would involve:
	// 1. Query peer0's DHT routing table for peer1's fingerprint
	// 2. Query peer1's DHT routing table for peer0's fingerprint
	// 3. Verify both peers can discover each other within timeout
	//
	// For Phase 1, we're just verifying the infrastructure works

	t.Log("NOTE: DHT discovery assertion not yet implemented (Phase 2)")
}
