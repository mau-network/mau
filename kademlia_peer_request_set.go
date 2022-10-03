package mau

import "sync"

// Sorted unique set of peers, adds one peer once
type peerRequestSet struct {
	mutex       sync.Mutex
	fingerprint Fingerprint
	peers       []*Peer
	added       map[Fingerprint]bool
}

func newPeerRequestSet(fpr Fingerprint, initial []*Peer) *peerRequestSet {
	peers := peerRequestSet{
		fingerprint: fpr,
		peers:       []*Peer{},
		added:       map[Fingerprint]bool{},
	}

	peers.add(initial...)

	return &peers
}

func (p *peerRequestSet) add(peers ...*Peer) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	for i := range peers {
		if p.added[peers[i].Fingerprint] {
			return
		}

		p.added[peers[i].Fingerprint] = true
		p.peers = append(p.peers, peers[i])
	}

	sortByDistance(p.fingerprint, p.peers)
}

func (p *peerRequestSet) get() *Peer {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.peers) == 0 {
		return nil
	}

	peer := p.peers[0]
	p.peers = p.peers[1:]
	return peer
}

func (p *peerRequestSet) len() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return len(p.peers)
}
