package mau

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/mdns"
)

// FingerprintResolver resolves a fingerprint into a IP:Port address
type FingerprintResolver func(ctx context.Context, fingerprint Fingerprint, addresses chan<- string) error

// StaticAddress returns the same address for any fingerprint
func StaticAddress(address string) FingerprintResolver {
	return func(ctx context.Context, fingerprint Fingerprint, addresses chan<- string) error {
		select {
		case <-ctx.Done():
			return nil
		default:
			addresses <- address
		}
		return nil
	}
}

// LocalFriendAddress resolves a fingerprint to the address of the friend if it
// was found on local network. it uses mDNS-SD to discover other peers on the
// local area network
func LocalFriendAddress(ctx context.Context, fingerprint Fingerprint, addresses chan<- string) error {
	name := fmt.Sprintf("%s.%s.%s.", fingerprint, mDNSServiceName, mDNSDomain)
	entriesCh := make(chan *mdns.ServiceEntry, 1)

	err := mdns.Lookup(mDNSServiceName, entriesCh)
	if err != nil {
		return err
	}

	for {
		select {
		case entry := <-entriesCh:
			if entry.Name == name {
				addresses <- fmt.Sprintf("%s:%d", entry.AddrV4, entry.Port)
				return nil
			}
		case <-ctx.Done():
			return nil
		}
	}
}

var ErrServerDoesNotAllowLookUp = errors.New("Server doesn't allow looking up friends on the internet")

// InternetFriendAddress returns a resolver function that will use the server
// kademlia network to lookup the friend address. it require the server to have
// already joined an overlay network. by having valid bootstrap peers already in
// the server when created.
func InternetFriendAddress(server *Server) FingerprintResolver {
	return func(ctx context.Context, fingerprint Fingerprint, addresses chan<- string) error {
		if server.dhtServer == nil {
			return ErrServerDoesNotAllowLookUp
		}

		// sendFindPeer needs to take a context to be able to cancel lookup
		peer := server.dhtServer.sendFindPeer(ctx, fingerprint)
		if peer != nil {
			addresses <- peer.Address
		}

		return nil
	}
}
