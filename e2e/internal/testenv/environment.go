package testenv

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/client"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	DefaultImage      = "mau-e2e:latest"
	NetworkNamePrefix = "mau-test"
	StateDir          = ".mau-e2e"
	StateFileName     = "current-env.json"
)

// Environment represents a test environment with multiple Mau peers
type Environment struct {
	Name        string       `json:"name"`
	NetworkName string       `json:"network_name"`
	NetworkID   string       `json:"network_id"`
	Peers       []*Peer      `json:"peers"`
	CreatedAt   time.Time    `json:"created_at"`
	
	dockerClient *client.Client
	network      testcontainers.Network
}

// Peer represents a single Mau peer container
type Peer struct {
	Name        string `json:"name"`
	ContainerID string `json:"container_id"`
	Fingerprint string `json:"fingerprint"`
	HTTPPort    int    `json:"http_port"`
	P2PPort     int    `json:"p2p_port"`
	Running     bool   `json:"running"`

	container testcontainers.Container
}

// NewEnvironment creates a new test environment
func NewEnvironment(name string) (*Environment, error) {
	if name == "" {
		name = fmt.Sprintf("mau-test-%d", time.Now().Unix())
	}

	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	env := &Environment{
		Name:         name,
		NetworkName:  fmt.Sprintf("%s-%s", NetworkNamePrefix, name),
		Peers:        []*Peer{},
		CreatedAt:    time.Now(),
		dockerClient: dockerClient,
	}

	return env, nil
}

// Start starts the environment with N peers
func (e *Environment) Start(ctx context.Context, numPeers int) error {
	// Create Docker network
	if err := e.createNetwork(ctx); err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	// Start peers
	for i := 0; i < numPeers; i++ {
		peerName := fmt.Sprintf("peer-%d", i)
		fmt.Printf("Starting peer %s...\n", peerName)

		peer, err := e.AddPeer(ctx, peerName)
		if err != nil {
			return fmt.Errorf("failed to start peer %s: %w", peerName, err)
		}

		fmt.Printf("  ✓ %s started (HTTP: %d, container: %s)\n",
			peer.Name, peer.HTTPPort, peer.ContainerID[:12])
	}

	// Save state
	if err := e.SaveState(); err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// createNetwork creates an isolated Docker network for the environment
func (e *Environment) createNetwork(ctx context.Context) error {
	networkReq := testcontainers.NetworkRequest{
		Name:           e.NetworkName,
		CheckDuplicate: true,
		Driver:         "bridge",
	}

	network, err := testcontainers.GenericNetwork(ctx, testcontainers.GenericNetworkRequest{
		NetworkRequest: networkReq,
	})
	if err != nil {
		return err
	}

	e.network = network
	// Get the actual network ID from Docker
	e.NetworkID = e.NetworkName
	return nil
}

// AddPeer adds a new peer to the environment
func (e *Environment) AddPeer(ctx context.Context, name string) (*Peer, error) {
	// Get Docker image from environment or use default
	image := os.Getenv("MAU_E2E_IMAGE")
	if image == "" {
		image = DefaultImage
	}

	// Create container request
	req := testcontainers.ContainerRequest{
		Image: image,
		Name:  fmt.Sprintf("%s-%s", e.Name, name),
		Networks: []string{e.NetworkName},
		ExposedPorts: []string{
			"8080/tcp", // HTTP port
			"9090/tcp", // P2P port
		},
		Env: map[string]string{
			"MAU_PEER_NAME": name,
			"MAU_PASSPHRASE": "testpass",
			"MAU_LOG_LEVEL": "info",
		},
		WaitingFor: wait.ForLog("Account:").WithStartupTimeout(30*time.Second),
		Labels: map[string]string{
			"mau-e2e-env": e.Name,
			"mau-e2e-peer": name,
		},
	}

	// Start container
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get container ID
	containerID := container.GetContainerID()

	// Get mapped ports
	httpPort, err := container.MappedPort(ctx, "8080")
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP port: %w", err)
	}

	p2pPort, err := container.MappedPort(ctx, "9090")
	if err != nil {
		return nil, fmt.Errorf("failed to get P2P port: %w", err)
	}

	// Get fingerprint (will be extracted from logs or API later)
	fingerprint, err := e.extractFingerprint(ctx, container)
	if err != nil {
		// Non-fatal, can be populated later
		fingerprint = "unknown"
	}

	peer := &Peer{
		Name:        name,
		ContainerID: containerID,
		Fingerprint: fingerprint,
		HTTPPort:    httpPort.Int(),
		P2PPort:     p2pPort.Int(),
		Running:     true,
		container:   container,
	}

	e.Peers = append(e.Peers, peer)

	return peer, nil
}

// extractFingerprint extracts the PGP fingerprint from the container
func (e *Environment) extractFingerprint(ctx context.Context, container testcontainers.Container) (string, error) {
	// TODO: Implement fingerprint extraction
	// Could be done by:
	// 1. Reading container logs for "Fingerprint: ..." output
	// 2. Calling Mau API endpoint to get account info
	// 3. Executing 'mau show' command in container
	
	// For now, return placeholder
	return "PLACEHOLDER", nil
}

// Stop stops the environment and cleans up resources
func (e *Environment) Stop(ctx context.Context) error {
	// Stop all peer containers
	for _, peer := range e.Peers {
		if peer.container != nil {
			fmt.Printf("Stopping peer %s...\n", peer.Name)
			if err := peer.container.Terminate(ctx); err != nil {
				fmt.Printf("  Warning: failed to stop %s: %v\n", peer.Name, err)
			}
		}
	}

	// Remove network
	if e.network != nil {
		fmt.Println("Removing network...")
		if err := e.network.Remove(ctx); err != nil {
			fmt.Printf("  Warning: failed to remove network: %v\n", err)
		}
	}

	// Delete state file
	if err := DeleteState(); err != nil {
		fmt.Printf("  Warning: failed to delete state: %v\n", err)
	}

	return nil
}

// FindPeer finds a peer by name
func (e *Environment) FindPeer(name string) *Peer {
	for _, peer := range e.Peers {
		if peer.Name == name {
			return peer
		}
	}
	return nil
}

// SaveState saves the environment state to disk
func (e *Environment) SaveState() error {
	stateDir := filepath.Join(os.Getenv("HOME"), StateDir)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	statePath := filepath.Join(stateDir, StateFileName)

	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// LoadCurrentEnvironment loads the current environment from state file
func LoadCurrentEnvironment() (*Environment, error) {
	stateDir := filepath.Join(os.Getenv("HOME"), StateDir)
	statePath := filepath.Join(stateDir, StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no active environment (state file not found)")
		}
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var env Environment
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("failed to parse state file: %w", err)
	}

	// Reconnect to Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}
	env.dockerClient = dockerClient

	// Reconnect to containers
	// (This is a simplified version; in production, we'd verify containers still exist)
	for range env.Peers {
		// Container handles will be nil, but container IDs are preserved
		// Commands that need container objects will need to re-acquire them
	}

	return &env, nil
}

// DeleteState deletes the state file
func DeleteState() error {
	stateDir := filepath.Join(os.Getenv("HOME"), StateDir)
	statePath := filepath.Join(stateDir, StateFileName)

	if err := os.Remove(statePath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// Logs returns a reader for the peer's container logs
func (p *Peer) Logs(ctx context.Context, follow bool, tail int) (io.ReadCloser, error) {
	// Note: This requires re-acquiring the container handle
	// In a full implementation, we'd store the container or use Docker API directly
	return nil, fmt.Errorf("logs not yet implemented - use 'docker logs %s'", p.ContainerID)
}
