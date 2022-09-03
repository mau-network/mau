package main

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/peerstore"
	"github.com/libp2p/go-libp2p-core/routing"
	"github.com/libp2p/go-libp2p/p2p/discovery"
	"github.com/multiformats/go-multiaddr"

	gostream "github.com/libp2p/go-libp2p-gostream"
	dht "github.com/libp2p/go-libp2p-kad-dht"
)

const protocolID = "/mau"
const discoveryID = "mau"

// TODO: implement peer functionality, try not to use specific libp2p premitives
type Peer struct {
	Account *Account
	context context.Context
	cancel  context.CancelFunc
	host    host.Host
}

func NewPeer(account *Account) (*Peer, error) {
	priv, _, err := crypto.KeyPairFromStdKey(account.entity.PrivateKey.PrivateKey)
	if err != nil {
		return nil, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	hst, err := libp2p.New(
		ctx,
		libp2p.ListenAddrStrings("/ip4/0.0.0.0/udp/0/quic"),
		// libp2p.Transport(libp2pquic.NewTransport),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
		libp2p.NATPortMap(),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			return dht.New(ctx, h)
		}),
	)
	if err != nil {
		cancel()
		return nil, err
	}

	peer := Peer{
		Account: account,
		context: ctx,
		cancel:  cancel,
		host:    hst,
	}

	return &peer, nil
}

func (p *Peer) Addresses() ([]string, error) {
	hostAddr, err := multiaddr.NewMultiaddr("/p2p/" + p.host.ID().Pretty())
	if err != nil {
		return nil, err
	}

	multiAddr := p.host.Addrs()
	addresses := []string{}
	for _, multiAddr := range multiAddr {
		addresses = append(addresses, multiAddr.Encapsulate(hostAddr).String())
	}
	return addresses, nil
}

func (p *Peer) Serve() error {
	listener, err := gostream.Listen(p.host, protocolID)
	gostream.Listen(p.host, protocolID)
	if err != nil {
		return err
	}
	defer listener.Close()

	server, err := NewServer(p.Account)
	if err != nil {
		return err
	}

	mdns, err := discovery.NewMdnsService(p.context, p.host, time.Second, discoveryID)
	if err != nil {
		return err
	}

	mdns.RegisterNotifee(p)

	return server.Serve(listener)
}

func (p *Peer) HandlePeerFound(pi peer.AddrInfo) {
	p.host.Peerstore().AddAddrs(pi.ID, pi.Addrs, peerstore.OwnObservedAddrTTL)
}
