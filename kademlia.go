package mau

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math/bits"
	"net/http"
	"net/url"
	"sort"
	"sync"
)

// KAD = Kademlia: A Peer-to-Peer Information System Based on the XOR Metric
const (
	DHT_B              = len(Fingerprint{}) * 8 // number of buckets
	DHT_K              = 20                     // max length of k bucket (replication parameter)
	DHT_ALPHA          = 3                      // parallelism factor
	DHT_PING_PATH      = "/kad/ping"
	DHT_FIND_NODE_PATH = "/kad/find_node"
)

type DHTNode struct {
	Fingerprint Fingerprint
	Address     string
}

// [],[],[],[],[],[],[],[]
// ^--Head (oldest)      ^--Tail (newest)
type bucket struct {
	mutex  sync.Mutex
	values []*DHTNode
}

// moveToTail moves a node that exists in the bucket to the end
func (b *bucket) moveToTail(node *DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]*DHTNode, 0, len(b.values))
	found := false
	for i := range b.values {
		if b.values[i].Fingerprint != node.Fingerprint {
			newValues = append(newValues, b.values[i])
		} else {
			found = true
		}
	}

	if found {
		b.values = append(newValues, node)
	}
}

// addToTail adds a node to the tail of the bucket
func (b *bucket) addToTail(node *DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.values = append(b.values, node)
}

// leastRecentlySeen returns the least recently seen node
func (b *bucket) leastRecentlySeen() *DHTNode {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.values) == 0 {
		return nil
	}

	return b.values[0]
}

// evict removes a node from the bucket
func (b *bucket) evict(node *DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]*DHTNode, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != node.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}
}

// Get returns a node from the bucket if fingerprint matches
func (b *bucket) get(node *DHTNode) *DHTNode {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for i := range b.values {
		if b.values[i].Fingerprint == node.Fingerprint {
			return b.values[i]
		}
	}

	return nil
}

func (b *bucket) dup() []*DHTNode {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	c := make([]*DHTNode, len(b.values))
	copy(c, b.values)

	return c
}

// isFull returns true if the bucket is full to the limit of K
func (b *bucket) isFull() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return len(b.values) < DHT_K
}

type routingTable struct {
	self    Fingerprint
	buckets [DHT_B]bucket
}

func (r *routingTable) bucketIndexFor(fingerprint Fingerprint) int {
	i := prefixLen(xor(r.self, fingerprint))
	if i == DHT_B {
		i--
	}
	return i
}

func (r *routingTable) bucketFor(fingerprint Fingerprint) *bucket {
	return &r.buckets[r.bucketIndexFor(fingerprint)]
}

// nearest returns list of nodes near fingerprint, limited to DHT_K
func (r *routingTable) nearest(fingerprint Fingerprint) []*DHTNode {
	b := r.bucketIndexFor(fingerprint) // nearest bucket
	nodes := r.buckets[b].dup()

	for i := 1; len(nodes) < DHT_K && (b-i >= 0 || b+i < DHT_B); i++ {
		if b-i >= 0 {
			nodes = append(nodes, r.buckets[b-i].values...)
		}
		if b+i < DHT_B {
			nodes = append(nodes, r.buckets[b+i].values...)
		}
	}

	sortByDistance(fingerprint, nodes)

	if len(nodes) > DHT_K {
		nodes = nodes[:DHT_K]
	}

	return nodes
}

// addNode adds a note to routing table if it doesn't exist if the bucket is
// full it pings the first node if the node responded it's discarded. else it
// removes the first node and adds the new node to the bucket
func (t *routingTable) addNode(node *DHTNode, ping func(*DHTNode) error) error {
	bucket := t.bucketFor(node.Fingerprint)

	if node := bucket.get(node); node != nil {
		bucket.moveToTail(node)
	} else if !bucket.isFull() {
		bucket.addToTail(node)
	} else if replacementNode := bucket.leastRecentlySeen(); replacementNode != nil {
		if ping(replacementNode) == nil {
			bucket.moveToTail(replacementNode)
		} else {
			bucket.evict(replacementNode)
			bucket.addToTail(node)
		}
	}

	return nil
}

func (t *routingTable) refreshStallBuckets() error {
	// TODO refresh stall buckets
	return nil
}

type DHTServer struct {
	mux          *http.ServeMux
	account      *Account
	routingTable routingTable
	Address      string
}

func NewDHTRPC(account *Account, address string) *DHTServer {
	d := &DHTServer{
		mux:     http.NewServeMux(),
		account: account,
		Address: address,
		routingTable: routingTable{
			self: account.Fingerprint(),
		},
	}

	d.mux.HandleFunc(DHT_PING_PATH, d.RecievePING)
	d.mux.HandleFunc(DHT_FIND_NODE_PATH, d.RecieveFIND_NODE)

	return d
}

func (d *DHTServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

// SendPING sends a ping to a node and returns true if the node response status isn't 2xx
//
// KAD (2.3)
func (d *DHTServer) SendPING(node *DHTNode) error {
	// TODO limit pinging a node in a period of time, we don't want to ping a
	// node multiple times per second for example, it's too much
	client, err := d.account.Client(node.Fingerprint, []string{d.Address})
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

// RecievePING KAD (2.3) responds with http.StatusOK
func (d *DHTServer) RecievePING(w http.ResponseWriter, r *http.Request) {
	d.addNode(r)
}

func (d *DHTServer) SendFIND_NODE(fingerprint Fingerprint) *DHTNode {
	nodes := d.routingTable.nearest(fingerprint)
	fingerprints := map[Fingerprint]bool{}
	for i := range nodes {
		if nodes[i].Fingerprint == fingerprint {
			return nodes[i]
		}
		fingerprints[nodes[i].Fingerprint] = true
	}

	var lock sync.Mutex
	var found *DHTNode
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

				client, err := d.account.Client(node.Fingerprint, []string{d.Address})
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
				d.routingTable.addNode(node, d.SendPING)

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

	return found
}

func (d *DHTServer) RecieveFIND_NODE(w http.ResponseWriter, r *http.Request) {
	d.addNode(r)

	fingerprint, err := ParseFingerprint(r.FormValue("fingerprint"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	nodes := d.routingTable.nearest(fingerprint)
	output, err := json.Marshal(nodes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(output)
}

// addNode: When any node send us a request add it to the contact list
// uses Post and IP headers as the node address.
// TODO review the idea of headers to get the peer address
func (d *DHTServer) addNode(r *http.Request) error {
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

	return d.routingTable.addNode(&node, d.SendPING)
}

// Join joins the network by adding a bootstrap known nodes to the routing table
// and querying about itself
func (d *DHTServer) Join(bootstrap []*DHTNode) error {
	// TODO
	return nil
}

// Refresh will refresh routing table stall buckets
func (d *DHTServer) Refresh() error {
	return d.routingTable.refreshStallBuckets()
}

// Binary operations

// xor two fingerprints
func xor(a, b Fingerprint) (c Fingerprint) {
	for i := range a {
		c[i] = a[i] ^ b[i]
	}

	return
}

// prefixLen returns the number of prefixed zero bits of a.
func prefixLen(a Fingerprint) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// sortByDistance sorts ids by descending XOR distance with respect to id.
func sortByDistance(id Fingerprint, nodes []*DHTNode) {
	sort.Slice(nodes, func(i, j int) bool {
		ixor := xor(nodes[i].Fingerprint, id)
		jxor := xor(nodes[j].Fingerprint, id)
		return bytes.Compare(ixor[:], jxor[:]) == -1
	})
}
