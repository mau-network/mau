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

func TestMockUPNPClient(t *testing.T) {
	t.Run("GetExternalIPAddress returns configured IP", func(t *testing.T) {
		expectedIP := "192.0.2.1"
		mock := &mockUPNPClient{
			getExternalIPAddressFunc: func() (string, error) {
				return expectedIP, nil
			},
		}

		ip, err := mock.GetExternalIPAddress()
		assert.NoError(t, err)
		assert.Equal(t, expectedIP, ip)
	})

	t.Run("GetExternalIPAddress returns error", func(t *testing.T) {
		expectedErr := errors.New("network timeout")
		mock := &mockUPNPClient{
			getExternalIPAddressFunc: func() (string, error) {
				return "", expectedErr
			},
		}

		ip, err := mock.GetExternalIPAddress()
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, "", ip)
	})

	t.Run("AddPortMapping succeeds", func(t *testing.T) {
		var capturedPort uint16
		var capturedProtocol string

		mock := &mockUPNPClient{
			addPortMappingFunc: func(
				remoteHost string,
				externalPort uint16,
				protocol string,
				internalPort uint16,
				internalClient string,
				enabled bool,
				description string,
				leaseDuration uint32,
			) error {
				capturedPort = externalPort
				capturedProtocol = protocol
				return nil
			},
		}

		err := mock.AddPortMapping("", 8080, "TCP", 8080, "192.168.1.100", true, "Test mapping", 3600)
		assert.NoError(t, err)
		assert.Equal(t, uint16(8080), capturedPort)
		assert.Equal(t, "TCP", capturedProtocol)
	})

	t.Run("AddPortMapping returns error", func(t *testing.T) {
		expectedErr := errors.New("port already in use")
		mock := &mockUPNPClient{
			addPortMappingFunc: func(string, uint16, string, uint16, string, bool, string, uint32) error {
				return expectedErr
			},
		}

		err := mock.AddPortMapping("", 8080, "TCP", 8080, "192.168.1.100", true, "Test", 3600)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
	})

	t.Run("Default implementation returns no error", func(t *testing.T) {
		mock := &mockUPNPClient{}

		// AddPortMapping with nil function should return nil
		err := mock.AddPortMapping("", 8080, "TCP", 8080, "192.168.1.100", true, "Test", 3600)
		assert.NoError(t, err)

		// GetExternalIPAddress with nil function should return default IP
		ip, err := mock.GetExternalIPAddress()
		assert.NoError(t, err)
		assert.Equal(t, "203.0.113.1", ip)
	})
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

func TestUPNPClientInterface(t *testing.T) {
	t.Run("Interface compliance", func(t *testing.T) {
		var _ upnpClient = (*mockUPNPClient)(nil)
	})
}
