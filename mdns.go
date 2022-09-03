package main

import (
	"io"

	"github.com/hashicorp/mdns"
)

const (
	MDNSServiceName = "_mau._tcp"
)

type MDNS struct {
	*mdns.Server
}

func (m *MDNS) Close() error {
	return m.Server.Shutdown()
}

func NewMDNSServer(account *Account, port int) (io.Closer, error) {
	fingerprint := account.Fingerprint()
	info := []string{fingerprint}

	service, err := mdns.NewMDNSService(fingerprint, MDNSServiceName, "", "", port, nil, info)
	if err != nil {
		return nil, err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	return &MDNS{server}, err
}
