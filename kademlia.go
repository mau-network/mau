package mau

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math/bits"
	"math/rand"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"time"
)

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric

const (
	dht_B                = FINGERPRINT_LEN * 8 // number of buckets
	dht_K                = 20                  // max length of k bucket (replication parameter)
	dht_ALPHA            = 3                   // parallelism factor
	dht_STALL_PERIOD     = time.Hour
	dht_PING_MIN_BACKOFF = 30 * time.Second // minimum time between pings to the same peer
)

// Peer is a reference to another instance of the program, identified by the
// address (host:port or ip:port) and Fingerprint of the public key. used for
// allowing the server to join a P2P network.
type Peer struct {
	Fingerprint Fingerprint `json:"fingerprint"`
	Address     string      `json:"address"` // Hostname:Port or IP:Port without the protocol
}

type dhtServer struct {
	mux           *http.ServeMux
	account       *Account
	address       string
	buckets       [dht_B]bucket
	cancelRefresh context.CancelFunc
	lastPing      map[Fingerprint]time.Time
	lastPingMutex sync.RWMutex
}

func newDHTServer(account *Account, address string) *dhtServer {
	d := &dhtServer{
		mux:      http.NewServeMux(),
		account:  account,
		address:  address,
		lastPing: make(map[Fingerprint]time.Time),
	}

	d.mux.HandleFunc("GET /kad/ping", d.receivePing)
	d.mux.HandleFunc("GET /kad/find_peer/{fpr}", d.receiveFindPeer)

	return d
}

func (d *dhtServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

// sendPing sends a ping to a peer and returns true if the peer response status isn't 2xx
func (d *dhtServer) sendPing(ctx context.Context, peer *Peer) error {
	// Rate limit pings to avoid excessive network traffic
	d.lastPingMutex.RLock()
	lastPingTime, exists := d.lastPing[peer.Fingerprint]
	d.lastPingMutex.RUnlock()

	if exists && time.Since(lastPingTime) < dht_PING_MIN_BACKOFF {
		// Too soon since last ping, return nil (success) to avoid removing peer
		return nil
	}

	client, err := d.account.Client(peer.Fingerprint, []string{d.address})
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: uriProtocolName,
		Host:   peer.Address,
		Path:   "/kad/ping",
	}

	_, err = client.client.R().SetContext(ctx).Get(u.String())

	if err == nil {
		// Update last ping time on success
		d.lastPingMutex.Lock()
		d.lastPing[peer.Fingerprint] = time.Now()
		d.lastPingMutex.Unlock()
	}

	return err
}

// receivePing responds with http.StatusOK
func (d *dhtServer) receivePing(w http.ResponseWriter, r *http.Request) {
	err := d.addPeerFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

// Lookup a peer by fingerprint
func (d *dhtServer) findPeerInNearest(nearest []*Peer, fingerprint Fingerprint) *Peer {
	for _, n := range nearest {
		if n.Fingerprint == fingerprint {
			return n
		}
	}
	return nil
}

func (d *dhtServer) runFindPeerWorker(ctx context.Context, fingerprint Fingerprint, peers *peerRequestSet, found **Peer, cancel context.CancelFunc) {
	for peers.len() > 0 && *found == nil {
		f, err := d.sendFindPeerWorker(ctx, fingerprint, peers)

		if err != nil {
			slog.Error("failed to find peer", "fingerprint", fingerprint, "error", err)
		}

		if f != nil {
			*found = f
			cancel()
			break
		}
	}
}

func (d *dhtServer) sendFindPeer(ctx context.Context, fingerprint Fingerprint) (found *Peer) {
	nearest := d.nearest(fingerprint, dht_ALPHA)
	nearest_count := len(nearest)

	if nearest_count == 0 {
		return nil
	}

	// if we already have it return it
	if existing := d.findPeerInNearest(nearest, fingerprint); existing != nil {
		return existing
	}

	peers := newPeerRequestSet(fingerprint, nearest)
	workersCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(nearest_count)

	for i := 0; i < nearest_count; i++ {
		go func() {
			d.runFindPeerWorker(workersCtx, fingerprint, peers, &found, cancel)
			wg.Done()
		}()
	}

	// if the passed context is canceled we cancel our workers
	go func() {
		<-ctx.Done()
		cancel()
	}()

	wg.Wait()
	return found
}

func (d *dhtServer) sendFindPeerWorker(ctx context.Context, fingerprint Fingerprint, peers *peerRequestSet) (*Peer, error) {
	peer := peers.get()
	if peer == nil {
		return nil, nil
	}

	client, err := d.account.Client(peer.Fingerprint, []string{d.address})
	if err != nil {
		return nil, err
	}

	var foundPeers []*Peer

	u := url.URL{
		Scheme: uriProtocolName,
		Host:   peer.Address,
		Path:   "/kad/find_peer/" + fingerprint.String(),
	}

	_, err = client.client.
		R().
		SetContext(ctx).
		ForceContentType("application/json").
		SetResult(&foundPeers).
		Get(u.String())

	if err != nil {
		d.removePeer(peer)
		return nil, err
	}

	// Add it to the known peers
	d.addPeer(peer)

	// peer should return max K peers, if it's more limit it to K
	if len(foundPeers) > dht_K {
		foundPeers = foundPeers[:dht_K]
	}

	for i := range foundPeers {
		if foundPeers[i].Fingerprint == fingerprint {
			return foundPeers[i], nil
		}
	}

	peers.add(foundPeers...)

	return nil, nil
}

func (d *dhtServer) receiveFindPeer(w http.ResponseWriter, r *http.Request) {
	err := d.addPeerFromRequest(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fpr := r.PathValue("fpr")
	fingerprint, err := FingerprintFromString(fpr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	peers := d.nearest(fingerprint, dht_K)
	output, err := json.Marshal(peers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := w.Write(output); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Join joins the network by adding a bootstrap known peers to the routing table
// and querying about itself
func (d *dhtServer) Join(ctx context.Context, bootstrap []*Peer) {
	if len(bootstrap) == 0 {
		return
	}

	for _, peer := range bootstrap {
		d.addPeer(peer)
	}

	d.sendFindPeer(ctx, d.account.Fingerprint())
	d.refreshAllBuckets(ctx)

	ctx, cancel := context.WithCancel(context.Background())
	go d.refreshStallBuckets(ctx)
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
	if r.TLS == nil {
		return ErrIncorrectPeerCertificate
	}

	fingerprint, err := FingerprintFromCert(r.TLS.PeerCertificates)
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

// addPeer adds a peer to routing table if it doesn't exist if the bucket is
// full it pings the first peer if the peer responded it's discarded. else it
// removes the first peer and adds the new peer to the bucket
func (d *dhtServer) addPeer(peer *Peer) {
	// don't add yourself to the contact list
	if peer.Fingerprint == d.account.Fingerprint() {
		return
	}

	bucket := &d.buckets[d.bucketFor(peer.Fingerprint)]

	if oldPeer := bucket.get(peer.Fingerprint); oldPeer != nil {
		bucket.moveToTail(oldPeer)
	} else if !bucket.isFull() {
		bucket.addToTail(peer)
	} else if existing := bucket.leastRecentlySeen(); existing != nil {
		if d.sendPing(context.Background(), existing) == nil {
			bucket.moveToTail(existing)
		} else {
			bucket.remove(existing)
			bucket.addToTail(peer)
		}
	}
}

func (d *dhtServer) removePeer(peer *Peer) {
	bucket := &d.buckets[d.bucketFor(peer.Fingerprint)]
	bucket.remove(peer)
}

// Refresh all stall buckets
func (d *dhtServer) refreshAllBuckets(ctx context.Context) {
	for i := range d.buckets {
		d.refreshBucket(ctx, i)
	}
}

// Refresh stall buckets
func (d *dhtServer) calculateNextRefreshTime(bucketIdx int, currentNextClick time.Duration) time.Duration {
	stallAfter := dht_STALL_PERIOD - time.Since(d.buckets[bucketIdx].lastLookup)
	if stallAfter < currentNextClick {
		return stallAfter
	}
	return currentNextClick
}

func (d *dhtServer) shouldRefreshBucket(bucketIdx int) bool {
	return time.Since(d.buckets[bucketIdx].lastLookup) >= dht_STALL_PERIOD
}

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
				if d.shouldRefreshBucket(i) {
					select {
					case <-ctx.Done():
						return
					default:
						d.refreshBucket(ctx, i)
					}
				} else {
					nextClick = d.calculateNextRefreshTime(i, nextClick)
				}
			}
		}
	}
}

func (d *dhtServer) refreshBucket(ctx context.Context, i int) {
	if rando := d.buckets[i].randomPeer(); rando != nil {
		d.removePeer(rando)
		d.sendFindPeer(ctx, rando.Fingerprint)
		d.addPeer(rando)
		d.buckets[i].lastLookup = time.Now()
	}
}

// nearest returns list of peers near fingerprint, limited to DHT_K
func (d *dhtServer) nearest(fingerprint Fingerprint, limit int) []*Peer {
	b := d.bucketFor(fingerprint) // nearest bucket
	peers := d.buckets[b].dup()

	for i := 1; len(peers) < limit && (b-i >= 0 || b+i < dht_B); i++ {
		if b-i >= 0 {
			peers = append(peers, d.buckets[b-i].values...)
		}
		if b+i < dht_B {
			peers = append(peers, d.buckets[b+i].values...)
		}
	}

	sortByDistance(fingerprint, peers)

	if len(peers) > limit {
		peers = peers[:limit]
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
			continue
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

// xor two fingerprints
func xor(a, b Fingerprint) (c Fingerprint) {
	for i := range a {
		c[i] = a[i] ^ b[i]
	}

	return
}

// prefixLen returns the number of leading zeros in a.
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
	slices.SortFunc(peers, func(a, b *Peer) int {
		ixor := xor(a.Fingerprint, fingerprint)
		jxor := xor(b.Fingerprint, fingerprint)
		return bytes.Compare(ixor[:], jxor[:])
	})
}
