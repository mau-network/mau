package mau

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/bits"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"time"
)

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric

const (
	dht_B              = len(Fingerprint{}) * 8 // number of buckets
	dht_K              = 20                     // max length of k bucket (replication parameter)
	dht_ALPHA          = 3                      // parallelism factor
	dht_STALL_PERIOD   = time.Hour
	dht_PING_PATH      = "/kad/ping"
	dht_FIND_PEER_PATH = "/kad/find_peer"
)

// Peer is a reference to another instance of the program, identified by the
// address (host:port or ip:port) and Fingerprint of the public key. used for
// allowing the server to join a P2P network.
type Peer struct {
	Fingerprint Fingerprint
	Address     string
}

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

	newValues := make([]*Peer, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != peer.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.values = newValues
	b.mutex.Unlock()
}

// addToTail adds a peer to the tail of the bucket
func (b *bucket) addToTail(peer *Peer) {
	b.mutex.Lock()
	b.values = append(b.values, peer)
	b.lastLookup = time.Now()
	b.mutex.Unlock()
}

// moveToTail moves a peer that exists in the bucket to the end
func (b *bucket) moveToTail(peer *Peer) {
	b.mutex.Lock()

	newValues := make([]*Peer, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != peer.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.values = append(newValues, peer)
	b.lastLookup = time.Now()
	b.mutex.Unlock()
}

// leastRecentlySeen returns the least recently seen peer
func (b *bucket) leastRecentlySeen() (peer *Peer) {
	b.mutex.RLock()

	if len(b.values) > 0 {
		peer = b.values[0]
	}

	b.mutex.RUnlock()

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

type dhtServer struct {
	mux           *http.ServeMux
	account       *Account
	address       string
	buckets       [dht_B]bucket
	cancelRefresh context.CancelFunc
}

func newDHTServer(account *Account, address string) *dhtServer {
	d := &dhtServer{
		mux:     http.NewServeMux(),
		account: account,
		address: address,
	}

	d.mux.HandleFunc(dht_PING_PATH, d.recievePing)
	d.mux.HandleFunc(dht_FIND_PEER_PATH, d.recieveFindPeer)

	return d
}

func (d *dhtServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

// sendPing sends a ping to a peer and returns true if the peer response status isn't 2xx
func (d *dhtServer) sendPing(peer *Peer) error {
	// TODO limit pinging a peer in a period of time, we don't want to ping a
	// peer multiple times per second for example, it's too much
	client, err := d.account.Client(peer.Fingerprint, []string{d.address})
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: uriProtocolName,
		Host:   peer.Address,
		Path:   dht_PING_PATH,
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	_, err = client.Do(req)

	return err
}

// recievePing responds with http.StatusOK
func (d *dhtServer) recievePing(_ http.ResponseWriter, r *http.Request) {
	d.addPeerFromRequest(r)
}

func (d *dhtServer) sendFindPeer(fingerprint Fingerprint) (found *Peer) {
	peers := d.nearest(fingerprint)
	fingerprints := map[Fingerprint]bool{}
	for i := range peers {
		if peers[i].Fingerprint == fingerprint {
			return peers[i]
		}
		fingerprints[peers[i].Fingerprint] = true
	}

	var lock sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	asked := map[Fingerprint]bool{}

	worker := func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:

				lock.Lock()

				if len(peers) == 0 {
					lock.Unlock()
					return
				}

				peer := peers[0]
				peers = peers[1:]

				delete(fingerprints, peer.Fingerprint)
				asked[peer.Fingerprint] = true
				lock.Unlock()

				client, err := d.account.Client(peer.Fingerprint, []string{d.address})
				if err != nil {
					break
				}

				u := url.URL{
					Scheme: uriProtocolName,
					Host:   peer.Address,
					Path:   dht_FIND_PEER_PATH,
				}

				req, err := http.NewRequest(http.MethodGet, u.String(), nil)
				if err != nil {
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				// Add it to the known peers
				d.addPeer(peer)

				foundPeers := []Peer{}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}

				if err = json.Unmarshal(body, &foundPeers); err != nil {
					continue
				}

				// peer should return max K peers, if it's more limit it to K
				if len(foundPeers) > dht_K {
					foundPeers = foundPeers[:dht_K]
				}

				lock.Lock()
				// Add all found peers that we don't have already and we didn't ask beofre
				for i := 0; i < len(foundPeers) && !fingerprints[foundPeers[i].Fingerprint] && !asked[peer.Fingerprint]; i++ {
					peers = append(peers, &foundPeers[i])
					fingerprints[foundPeers[i].Fingerprint] = true
					if foundPeers[i].Fingerprint == fingerprint {
						found = &foundPeers[i]
						cancel()
						break
					}
				}
				sortByDistance(fingerprint, peers)
				lock.Unlock()
			}
		}
	}

	for i := 0; i < dht_ALPHA; i++ {
		go worker()
	}

	return
}

func (d *dhtServer) recieveFindPeer(w http.ResponseWriter, r *http.Request) {
	d.addPeerFromRequest(r)

	fingerprint, err := ParseFingerprint(r.FormValue("fingerprint"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	peers := d.nearest(fingerprint)
	output, err := json.Marshal(peers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(output)
}

// Join joins the network by adding a bootstrap known peers to the routing table
// and querying about itself
func (d *dhtServer) Join(bootstrap []*Peer) {
	if len(bootstrap) == 0 {
		return
	}

	for _, peer := range bootstrap {
		d.addPeer(peer)
	}

	d.sendFindPeer(d.account.Fingerprint())
	d.refreshAllBuckets()

	ctx, cancel := context.WithCancel(context.Background())
	d.refreshStallBuckets(ctx)
	d.cancelRefresh = cancel
}

// Leave terminates any background jobs
func (d *dhtServer) Leave() {
	if d.cancelRefresh != nil {
		d.cancelRefresh()
	}
}

// addPeerFromRequest: When any peers send us a request add it to the contact list
func (d *dhtServer) addPeerFromRequest(r *http.Request) error {
	fingerprint, err := certToFingerprint(r.TLS.PeerCertificates)
	if err != nil {
		return err
	}

	address, err := certToAddress(r.TLS.PeerCertificates)
	if err != nil {
		return err
	}

	d.addPeer(&Peer{
		Fingerprint: fingerprint,
		Address:     address,
	})

	return nil
}

// addPeer adds a note to routing table if it doesn't exist if the bucket is
// full it pings the first peer if the peer responded it's discarded. else it
// removes the first peer and adds the new peer to the bucket
func (d *dhtServer) addPeer(peer *Peer) {
	bucket := &d.buckets[d.bucketFor(peer.Fingerprint)]

	if oldPeer := bucket.get(peer.Fingerprint); oldPeer != nil {
		bucket.moveToTail(oldPeer)
	} else if !bucket.isFull() {
		bucket.addToTail(peer)
	} else if existing := bucket.leastRecentlySeen(); existing != nil {
		if d.sendPing(existing) == nil {
			bucket.moveToTail(existing)
		} else {
			bucket.remove(existing)
			bucket.addToTail(peer)
		}
	}
}

// Refresh all stall buckets
func (d *dhtServer) refreshAllBuckets() {
	for i := range d.buckets {
		d.refreshBucket(i)
	}
}

// Refresh stall buckets
func (d *dhtServer) refreshStallBuckets(ctx context.Context) {
	nextClick := time.Duration(0)

	// refresh the buckets indefinitely
	for {
		nextClick = dht_STALL_PERIOD

		// either the context is done and we exit of the next click trigger refreshing buckets
		select {
		case <-ctx.Done():
			return
		case <-time.After(nextClick):

			// we'll go over all buckets and refresh the bucket or exit
			for i := range d.buckets {
				// if it's refreshable we'll refresh it
				if time.Since(d.buckets[i].lastLookup) > dht_STALL_PERIOD {
					select {
					case <-ctx.Done():
						return
					default:
						d.refreshBucket(i)
					}

				} else {
					// if it's not refreshable then calculate when it's gonna
					// need refresh and move next click earlier if needed
					stallAfter := dht_STALL_PERIOD - time.Now().Sub(d.buckets[i].lastLookup)
					if stallAfter < nextClick {
						nextClick = stallAfter
					}
				}
			}

		}
	}
}

// TODO add context for faster termination
func (d *dhtServer) refreshBucket(i int) {
	if rando := d.buckets[i].randomPeer(); rando != nil {
		d.sendFindPeer(rando.Fingerprint)
		d.buckets[i].lastLookup = time.Now()
	}
}

// nearest returns list of peers near fingerprint, limited to DHT_K
func (d *dhtServer) nearest(fingerprint Fingerprint) []*Peer {
	b := d.bucketFor(fingerprint) // nearest bucket
	peers := d.buckets[b].dup()

	for i := 1; len(peers) < dht_K && (b-i >= 0 || b+i < dht_B); i++ {
		if b-i >= 0 {
			peers = append(peers, d.buckets[b-i].values...)
		}
		if b+i < dht_B {
			peers = append(peers, d.buckets[b+i].values...)
		}
	}

	sortByDistance(fingerprint, peers)

	if len(peers) > dht_K {
		peers = peers[:dht_K]
	}

	return peers
}

// bucketFor returns the Index of the bucket this fingerprint belongs to
func (d *dhtServer) bucketFor(fingerprint Fingerprint) (i int) {
	i = prefixLen(xor(d.account.Fingerprint(), fingerprint))
	if i == dht_B {
		i--
	}
	return
}

// Binary operations

// xor two fingerprints
func xor(a, b Fingerprint) (c Fingerprint) {
	for i := range a {
		c[i] = a[i] ^ b[i]
	}

	return
}

// prefixLen returns the number of leading zeros in a.
// TODO try the log2 approach to get leading zeros
func prefixLen(a Fingerprint) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// sortByDistance sorts ids by ascending XOR distance with respect to fingerprint
func sortByDistance(fingerprint Fingerprint, peers []*Peer) {
	sort.Slice(peers, func(i, j int) bool {
		ixor := xor(peers[i].Fingerprint, fingerprint)
		jxor := xor(peers[j].Fingerprint, fingerprint)
		return bytes.Compare(ixor[:], jxor[:]) == -1
	})
}
