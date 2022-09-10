package main

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

func UPNPFactory[T UPNPClient](f func() ([]T, []error, error)) func() []UPNPClient {
	return func() []UPNPClient {
		r, _, _ := f()
		cs := make([]UPNPClient, 0, len(r))
		for _, i := range r {
			cs = append(cs, i)
		}
		return cs
	}
}

// TODO This function doesn't return clients if the firewall is enabled. find a way to ask the firewall for port
func NewUPNPClient(ctx context.Context) (UPNPClient, error) {
	funcs := []func() []UPNPClient{
		UPNPFactory(internetgateway2.NewWANIPConnection1Clients),
		UPNPFactory(internetgateway2.NewWANIPConnection2Clients),
		UPNPFactory(internetgateway2.NewWANPPPConnection1Clients),
		UPNPFactory(internetgateway1.NewWANIPConnection1Clients),
		UPNPFactory(internetgateway1.NewWANPPPConnection1Clients),
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
