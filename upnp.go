package mau

import (
	"context"
	"errors"

	"github.com/huin/goupnp/dcps/internetgateway1"
	"github.com/huin/goupnp/dcps/internetgateway2"
)

type UPNPClient interface {
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

// TODO This function doesn't return clients if the firewall is enabled. find a way to ask the firewall for port
func NewUPNPClient(ctx context.Context) (UPNPClient, error) {
	funcs := []func() []UPNPClient{
		upnpFactory(internetgateway2.NewWANIPConnection1Clients),
		upnpFactory(internetgateway2.NewWANIPConnection2Clients),
		upnpFactory(internetgateway2.NewWANPPPConnection1Clients),
		upnpFactory(internetgateway1.NewWANIPConnection1Clients),
		upnpFactory(internetgateway1.NewWANPPPConnection1Clients),
	}

	for _, f := range funcs {
		if cs := f(); len(cs) > 0 {
			for _, c := range cs {
				return c, nil
			}
		}
	}

	return nil, errors.New("No services found. Please make sure the firewall is not blocking connections.")
}

func upnpFactory[T UPNPClient](f func() ([]T, []error, error)) func() []UPNPClient {
	return func() []UPNPClient {
		r, _, _ := f()
		cs := make([]UPNPClient, 0, len(r))
		for _, i := range r {
			cs = append(cs, i)
		}
		return cs
	}
}
