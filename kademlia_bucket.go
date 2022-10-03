package mau

import (
	"math/rand"
	"sync"
	"time"
)

// A list of peers
// [],[],[],[],[],[],[],[]
// ^--Head (oldest)      ^--Tail (newest)
type bucket struct {
	mutex      sync.RWMutex
	values     []*Peer
	lastLookup time.Time
}

// get returns a peer from the bucket by fingerprint
func (b *bucket) get(fingerprint Fingerprint) *Peer {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for i := range b.values {
		if b.values[i].Fingerprint == fingerprint {
			return b.values[i]
		}
	}

	return nil
}

// remove removes a peer from the bucket
func (b *bucket) remove(peer *Peer) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]*Peer, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != peer.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.values = newValues
}

// addToTail adds a peer to the tail of the bucket
func (b *bucket) addToTail(peer *Peer) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.values = append(b.values, peer)
	b.lastLookup = time.Now()
}

// moveToTail moves a peer that exists in the bucket to the end
func (b *bucket) moveToTail(peer *Peer) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]*Peer, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != peer.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.values = append(newValues, peer)
	b.lastLookup = time.Now()
}

// leastRecentlySeen returns the least recently seen peer
func (b *bucket) leastRecentlySeen() (peer *Peer) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if len(b.values) > 0 {
		peer = b.values[0]
	}

	return
}

// randomPeer returns a random peer from the bucket
func (b *bucket) randomPeer() *Peer {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	if len(b.values) == 0 {
		return nil
	}

	return b.values[rand.Intn(len(b.values))]
}

// isFull returns true if the bucket is full to the limit of K
func (b *bucket) isFull() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return len(b.values) == dht_K
}

// dup returns a copy of the bucket values
func (b *bucket) dup() []*Peer {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	c := make([]*Peer, len(b.values))
	copy(c, b.values)

	return c
}
