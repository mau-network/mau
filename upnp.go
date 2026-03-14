package mau

import (
	"context"
	"errors"

	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
)

type upnpClient interface {
	AddPortMapping(
		NewRemoteHost string,
		NewExternalPort uint16,
		NewProtocol string,
		NewInternalPort uint16,
		NewInternalClient string,
		NewEnabled bool,
		NewPortMappingDescription string,
		NewLeaseDuration uint32,
	) (err error)

	GetExternalIPAddress() (
		NewExternalIPAddress string,
		err error,
	)
}

// newUPNPClient discovers UPnP-enabled Internet Gateway Devices (IGDs) on the local network.
// It tries multiple service types in order of preference (IGDv2 first, then IGDv1).
// The context allows for timeout control during network discovery.
//
// Returns the first available client or an error if no services are found.
// Note: Discovery may fail if the firewall blocks UPnP/SSDP multicast packets (UDP 1900).
func newUPNPClient(ctx context.Context) (upnpClient, error) {
	funcs := []func(context.Context) []upnpClient{
		upnpFactoryCtx(internetgateway2.NewWANIPConnection1ClientsCtx),
		upnpFactoryCtx(internetgateway2.NewWANIPConnection2ClientsCtx),
		upnpFactoryCtx(internetgateway2.NewWANPPPConnection1ClientsCtx),
		upnpFactoryCtx(internetgateway1.NewWANIPConnection1ClientsCtx),
		upnpFactoryCtx(internetgateway1.NewWANPPPConnection1ClientsCtx),
	}

	for _, f := range funcs {
		if cs := f(ctx); len(cs) > 0 {
			for _, c := range cs {
				return c, nil
			}
		}
	}

	return nil, errors.New("No services found. Please make sure the firewall is not blocking connections.")
}

// upnpFactoryCtx adapts context-aware UPnP client discovery functions to the upnpClient interface.
// It wraps the library's typed client discovery to return a slice of interface implementations.
func upnpFactoryCtx[T upnpClient](f func(context.Context) ([]T, []error, error)) func(context.Context) []upnpClient {
	return func(ctx context.Context) []upnpClient {
		r, _, _ := f(ctx)
		cs := make([]upnpClient, 0, len(r))
		for _, i := range r {
			cs = append(cs, i)
		}
		return cs
	}
}

// upnpFactory adapts legacy (non-context) UPnP client discovery functions to the upnpClient interface.
// Deprecated: Use upnpFactoryCtx for proper context support.
func upnpFactory[T upnpClient](f func() ([]T, []error, error)) func() []upnpClient {
	return func() []upnpClient {
		r, _, _ := f()
		cs := make([]upnpClient, 0, len(r))
		for _, i := range r {
			cs = append(cs, i)
		}
		return cs
	}
}
