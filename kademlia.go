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
	DHT_B              = len(Fingerprint{}) * 8 // number of buckets
	DHT_K              = 20                     // max length of k bucket (replication parameter)
	DHT_ALPHA          = 3                      // parallelism factor
	DHT_STALL_PERIOD   = time.Hour
	DHT_PING_PATH      = "/kad/ping"
	DHT_FIND_NODE_PATH = "/kad/find_node"
)

type DHTNode struct {
	Fingerprint Fingerprint
	Address     string
}

// A list of nodes
// [],[],[],[],[],[],[],[]
// ^--Head (oldest)      ^--Tail (newest)
type bucket struct {
	mutex      sync.RWMutex
	values     []*DHTNode
	lastLookup time.Time
}

// get returns a node from the bucket by fingerprint
func (b *bucket) get(fingerprint Fingerprint) *DHTNode {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	for i := range b.values {
		if b.values[i].Fingerprint == fingerprint {
			return b.values[i]
		}
	}

	return nil
}

// remove removes a node from the bucket
func (b *bucket) remove(node *DHTNode) {
	b.mutex.Lock()

	newValues := make([]*DHTNode, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != node.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.mutex.Unlock()
}

// addToTail adds a node to the tail of the bucket
func (b *bucket) addToTail(node *DHTNode) {
	b.mutex.Lock()
	b.values = append(b.values, node)
	b.lastLookup = time.Now()
	b.mutex.Unlock()
}

// moveToTail moves a node that exists in the bucket to the end
func (b *bucket) moveToTail(node *DHTNode) {
	b.mutex.Lock()

	newValues := make([]*DHTNode, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != node.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}

	b.values = append(newValues, node)
	b.lastLookup = time.Now()
	b.mutex.Unlock()
}

// leastRecentlySeen returns the least recently seen node
func (b *bucket) leastRecentlySeen() (node *DHTNode) {
	b.mutex.RLock()

	if len(b.values) > 0 {
		node = b.values[0]
	}

	b.mutex.RUnlock()

	return
}

// isFull returns true if the bucket is full to the limit of K
func (b *bucket) isFull() bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return len(b.values) < DHT_K
}

// dup returns a copy of the bucket values
func (b *bucket) dup() []*DHTNode {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	c := make([]*DHTNode, len(b.values))
	copy(c, b.values)

	return c
}

func (b *bucket) randomNode() *DHTNode {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	return b.values[rand.Intn(len(b.values))]
}

type DHTServer struct {
	mux           *http.ServeMux
	account       *Account
	address       string
	buckets       [DHT_B]bucket
	cancelRefresh context.CancelFunc
}

func NewDHTRPC(account *Account, address string) *DHTServer {
	d := &DHTServer{
		mux:     http.NewServeMux(),
		account: account,
		address: address,
	}

	d.mux.HandleFunc(DHT_PING_PATH, d.RecievePing)
	d.mux.HandleFunc(DHT_FIND_NODE_PATH, d.RecieveFindNode)

	return d
}

func (d *DHTServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

// SendPing sends a ping to a node and returns true if the node response status isn't 2xx
func (d *DHTServer) SendPing(node *DHTNode) error {
	// TODO limit pinging a node in a period of time, we don't want to ping a
	// node multiple times per second for example, it's too much
	client, err := d.account.Client(node.Fingerprint, []string{d.address})
	if err != nil {
		return err
	}

	u := url.URL{
		Scheme: uriProtocolName,
		Host:   node.Address,
		Path:   DHT_PING_PATH,
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	_, err = client.Do(req)

	return err
}

// RecievePing responds with http.StatusOK
func (d *DHTServer) RecievePing(_ http.ResponseWriter, r *http.Request) {
	d.addNodeFromRequest(r)
}

func (d *DHTServer) SendFindNode(fingerprint Fingerprint) (found *DHTNode) {
	nodes := d.nearest(fingerprint)
	fingerprints := map[Fingerprint]bool{}
	for i := range nodes {
		if nodes[i].Fingerprint == fingerprint {
			return nodes[i]
		}
		fingerprints[nodes[i].Fingerprint] = true
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

				if len(nodes) == 0 {
					lock.Unlock()
					return
				}

				node := nodes[0]
				nodes = nodes[1:]

				delete(fingerprints, node.Fingerprint)
				asked[node.Fingerprint] = true
				lock.Unlock()

				client, err := d.account.Client(node.Fingerprint, []string{d.address})
				if err != nil {
					break
				}

				u := url.URL{
					Scheme: uriProtocolName,
					Host:   node.Address,
					Path:   DHT_FIND_NODE_PATH,
				}

				req, err := http.NewRequest(http.MethodGet, u.String(), nil)
				if err != nil {
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					continue
				}

				// Add it to the known nodes
				d.addNode(node)

				foundNodes := []DHTNode{}
				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				if err != nil {
					continue
				}

				if err = json.Unmarshal(body, &foundNodes); err != nil {
					continue
				}

				// peer should return max K node, if it's more limit it to K
				if len(foundNodes) > DHT_K {
					foundNodes = foundNodes[:DHT_K]
				}

				lock.Lock()
				// Add all found nodes that we don't have already and we didn't ask beofre
				for i := 0; i < len(foundNodes) && !fingerprints[foundNodes[i].Fingerprint] && !asked[node.Fingerprint]; i++ {
					nodes = append(nodes, &foundNodes[i])
					fingerprints[foundNodes[i].Fingerprint] = true
					if foundNodes[i].Fingerprint == fingerprint {
						found = &foundNodes[i]
						cancel()
						break
					}
				}
				sortByDistance(fingerprint, nodes)
				lock.Unlock()
			}
		}
	}

	for i := 0; i < DHT_ALPHA; i++ {
		go worker()
	}

	return
}

func (d *DHTServer) RecieveFindNode(w http.ResponseWriter, r *http.Request) {
	d.addNodeFromRequest(r)

	fingerprint, err := ParseFingerprint(r.FormValue("fingerprint"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	nodes := d.nearest(fingerprint)
	output, err := json.Marshal(nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(output)
}

// Join joins the network by adding a bootstrap known nodes to the routing table
// and querying about itself
func (d *DHTServer) Join(bootstrap []*DHTNode) {
	for _, node := range bootstrap {
		d.addNode(node)
	}

	d.SendFindNode(d.account.Fingerprint())
	d.refreshAllBuckets()

	ctx, cancel := context.WithCancel(context.Background())
	d.refreshStallBuckets(ctx)
	d.cancelRefresh = cancel
}

// Leave terminates any background jobs
func (d *DHTServer) Leave() {
	if d.cancelRefresh != nil {
		d.cancelRefresh()
	}
}

// addNode: When any node send us a request add it to the contact list
// uses Post and IP headers as the node address.
func (d *DHTServer) addNodeFromRequest(r *http.Request) error {
	fingerprint, err := certToFingerprint(r.TLS.PeerCertificates)
	if err != nil {
		return err
	}

	address, err := certToAddress(r.TLS.PeerCertificates)
	if err != nil {
		return err
	}

	node := DHTNode{
		Fingerprint: fingerprint,
		Address:     address,
	}

	d.addNode(&node)

	return nil
}

// Refresh all buckets
func (d *DHTServer) refreshAllBuckets() {
	for i := range d.buckets {
		d.refreshBucket(i)
	}
}

// Refresh stall buckets
func (d *DHTServer) refreshStallBuckets(ctx context.Context) {
	nextClick := time.Duration(0)

	// refresh the buckets indefinitely
	for {
		nextClick = DHT_STALL_PERIOD

		// either the context is done and we exit of the next click trigger refreshing buckets
		select {
		case <-ctx.Done():
			return
		case <-time.After(nextClick):

			// we'll go over all buckets and refresh the bucket or exit
			for i := range d.buckets {
				// if it's refreshable we'll refresh it
				if time.Since(d.buckets[i].lastLookup) > DHT_STALL_PERIOD {
					select {
					case <-ctx.Done():
						return
					default:
						d.refreshBucket(i)
					}

				} else {
					// if it's not refreshable then calculate when it's gonna
					// need refresh and move next click earlier if needed
					stallAfter := DHT_STALL_PERIOD - time.Now().Sub(d.buckets[i].lastLookup)
					if stallAfter < nextClick {
						nextClick = stallAfter
					}
				}
			}

		}
	}
}

// TODO add context for faster termination
func (d *DHTServer) refreshBucket(i int) {
	if rando := d.buckets[i].randomNode(); rando != nil {
		d.SendFindNode(rando.Fingerprint)
		d.buckets[i].lastLookup = time.Now()
	}
}

// nearest returns list of nodes near fingerprint, limited to DHT_K
func (d *DHTServer) nearest(fingerprint Fingerprint) []*DHTNode {
	b := d.bucketFor(fingerprint) // nearest bucket
	nodes := d.buckets[b].dup()

	for i := 1; len(nodes) < DHT_K && (b-i >= 0 || b+i < DHT_B); i++ {
		if b-i >= 0 {
			nodes = append(nodes, d.buckets[b-i].values...)
		}
		if b+i < DHT_B {
			nodes = append(nodes, d.buckets[b+i].values...)
		}
	}

	sortByDistance(fingerprint, nodes)

	if len(nodes) > DHT_K {
		nodes = nodes[:DHT_K]
	}

	return nodes
}

// bucketFor returns the Index of the bucket this fingerprint belongs to
func (d *DHTServer) bucketFor(fingerprint Fingerprint) (i int) {
	i = prefixLen(xor(d.account.Fingerprint(), fingerprint))
	if i == DHT_B {
		i--
	}
	return
}

// addNode adds a note to routing table if it doesn't exist if the bucket is
// full it pings the first node if the node responded it's discarded. else it
// removes the first node and adds the new node to the bucket
func (d *DHTServer) addNode(node *DHTNode) {
	bucket := &d.buckets[d.bucketFor(node.Fingerprint)]

	if oldNode := bucket.get(node.Fingerprint); oldNode != nil {
		bucket.moveToTail(oldNode)
	} else if !bucket.isFull() {
		bucket.addToTail(node)
	} else if existing := bucket.leastRecentlySeen(); existing != nil {
		if d.SendPing(existing) == nil {
			bucket.moveToTail(existing)
		} else {
			bucket.remove(existing)
			bucket.addToTail(node)
		}
	}
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
func prefixLen(a Fingerprint) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// sortByDistance sorts ids by ascending XOR distance with respect to fingerprint
func sortByDistance(fingerprint Fingerprint, nodes []*DHTNode) {
	sort.Slice(nodes, func(i, j int) bool {
		ixor := xor(nodes[i].Fingerprint, fingerprint)
		jxor := xor(nodes[j].Fingerprint, fingerprint)
		return bytes.Compare(ixor[:], jxor[:]) == -1
	})
}
