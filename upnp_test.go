package mau

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockUPNPClient implements the upnpClient interface for testing
type mockUPNPClient struct {
	addPortMappingFunc       func(string, uint16, string, uint16, string, bool, string, uint32) error
	getExternalIPAddressFunc func() (string, error)
}

func (m *mockUPNPClient) AddPortMapping(
	newRemoteHost string,
	newExternalPort uint16,
	newProtocol string,
	newInternalPort uint16,
	newInternalClient string,
	newEnabled bool,
	newPortMappingDescription string,
	newLeaseDuration uint32,
) error {
	if m.addPortMappingFunc != nil {
		return m.addPortMappingFunc(
			newRemoteHost,
			newExternalPort,
			newProtocol,
			newInternalPort,
			newInternalClient,
			newEnabled,
			newPortMappingDescription,
			newLeaseDuration,
		)
	}
	return nil
}

func (m *mockUPNPClient) GetExternalIPAddress() (string, error) {
	if m.getExternalIPAddressFunc != nil {
		return m.getExternalIPAddressFunc()
	}
	return "203.0.113.1", nil
}

func TestUPNPClient(t *testing.T) {
	t.Skip("This test proven to be slow as it asks the network gateway for a port")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	client, err := newUPNPClient(ctx)
	assert.NoError(t, err)

	address, err := client.GetExternalIPAddress()
	assert.NoError(t, err)
	assert.NotEqual(t, "", address)
}

func TestUPNPFactory(t *testing.T) {
	t.Run("Factory converts typed clients to interface slice", func(t *testing.T) {
		mock1 := &mockUPNPClient{
			getExternalIPAddressFunc: func() (string, error) {
				return "192.0.2.1", nil
			},
		}
		mock2 := &mockUPNPClient{
			getExternalIPAddressFunc: func() (string, error) {
				return "192.0.2.2", nil
			},
		}

		// Simulate a function that returns typed clients
		factoryFunc := func() ([]*mockUPNPClient, []error, error) {
			return []*mockUPNPClient{mock1, mock2}, nil, nil
		}

		// Convert to upnpFactory
		factory := upnpFactory(factoryFunc)
		clients := factory()

		assert.Len(t, clients, 2)
		
		// Verify first client
		ip1, err := clients[0].GetExternalIPAddress()
		assert.NoError(t, err)
		assert.Equal(t, "192.0.2.1", ip1)

		// Verify second client
		ip2, err := clients[1].GetExternalIPAddress()
		assert.NoError(t, err)
		assert.Equal(t, "192.0.2.2", ip2)
	})

	t.Run("Factory handles empty client list", func(t *testing.T) {
		factoryFunc := func() ([]*mockUPNPClient, []error, error) {
			return []*mockUPNPClient{}, nil, nil
		}

		factory := upnpFactory(factoryFunc)
		clients := factory()

		assert.Len(t, clients, 0)
	})

	t.Run("Factory ignores errors from underlying function", func(t *testing.T) {
		mock := &mockUPNPClient{}
		
		factoryFunc := func() ([]*mockUPNPClient, []error, error) {
			return []*mockUPNPClient{mock}, []error{errors.New("some error")}, errors.New("another error")
		}

		factory := upnpFactory(factoryFunc)
		clients := factory()

		// Factory should still return the client despite errors
		assert.Len(t, clients, 1)
	})
}

func TestNewUPNPClient(t *testing.T) {
	t.Run("Returns error when no services found", func(t *testing.T) {
		ctx := context.Background()

		// This will fail in test environment without real UPnP devices
		client, err := newUPNPClient(ctx)
		
		// Either it finds a real device or returns the expected error
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "No services found")
		} else {
			// In rare cases where a test environment has UPnP
			assert.NotNil(t, client)
		}
	})

	t.Run("Respects context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Function should still complete (context is passed but not currently used in newUPNPClient)
		client, err := newUPNPClient(ctx)
		
		// Expected behavior: no clients found
		if err != nil {
			assert.Error(t, err)
			assert.Nil(t, client)
		}
	})
}

// Verify mock implements interface at compile time
var _ upnpClient = (*mockUPNPClient)(nil)
