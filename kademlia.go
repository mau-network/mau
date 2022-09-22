package mau

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
)

const (
	DHT_B              = len(Fingerprint{}) * 8 // number of buckets
	DHT_K              = 20                     // max length of k bucket (replication parameter)
	DHT_ALPHA          = 3                      // parallelism factor
	DHT_PING_PATH      = "/kad/ping"
	DHT_FIND_NODE_PATH = "/kad/find_node"
)

type DHTNode struct {
	Fingerprint Fingerprint
	IP          net.IP
	Port        int
}

func (d *DHTNode) Address() string {
	return fmt.Sprintf("%s%s:%d", uriProtocolName, d.IP, d.Port)
}

// [],[],[],[],[],[],[],[]
// ^--Head (oldest)      ^--Tail (newest)
type bucket struct {
	mutex  sync.Mutex
	values []DHTNode
}

// moveToTail moves a node that exists in the bucket to the end
func (b *bucket) moveToTail(node DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]DHTNode, 0, len(b.values))
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
func (b *bucket) addToTail(node DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	b.values = append(b.values, node)
}

// leastRecentlySeen returns the least recently seen node. if the bucket is
// empty return io.EOF error
func (b *bucket) leastRecentlySeen() (DHTNode, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	if len(b.values) == 0 {
		return DHTNode{}, io.EOF
	}

	return b.values[0], nil
}

// evict removes a node from the bucket
func (b *bucket) evict(node DHTNode) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	newValues := make([]DHTNode, 0, len(b.values))
	for i := range b.values {
		if b.values[i].Fingerprint != node.Fingerprint {
			newValues = append(newValues, b.values[i])
		}
	}
}

// Get returns a node from the bucket if fingerprint matches
func (b *bucket) get(node DHTNode) (DHTNode, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	for i := range b.values {
		if b.values[i].Fingerprint == node.Fingerprint {
			return b.values[i], nil
		}
	}

	return DHTNode{}, io.EOF
}

// isFull returns true if the bucket is full to the limit of K
func (b *bucket) isFull() bool {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	return len(b.values) < DHT_K
}

type routingTable [DHT_B]bucket

func (r *routingTable) bucketFor(node DHTNode) *bucket {
	// TODO
	return &r[0]
}

func (r *routingTable) nearest(fingerprint Fingerprint) []DHTNode {
	// TODO
	return []DHTNode{}
}

// addNode adds a note to routing table if it doesn't exist if the bucket is
// full it pings the first node if the node responded it's discarded. else it
// removes the first node and adds the new node to the bucket
func (t *routingTable) addNode(node DHTNode, ping func(DHTNode) error) {
	bucket := t.bucketFor(node)

	if node, err := bucket.get(node); err != nil {
		bucket.moveToTail(node)
	} else if bucket.isFull() {
		bucket.addToTail(node)
	} else {
		replacementNode, err := bucket.leastRecentlySeen()
		if err == nil && ping(replacementNode) == nil {
			bucket.moveToTail(replacementNode)
		} else {
			bucket.evict(replacementNode)
			bucket.addToTail(node)
		}
	}
}

type DHTServer struct {
	mux          *http.ServeMux
	account      *Account
	routingTable routingTable
}

// KAD = Kademlia: A Peer-to-Peer Information System Based on the XOR Metric
func NewDHTRPC(account *Account) *DHTServer {
	d := &DHTServer{
		mux:     http.NewServeMux(),
		account: account,
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
func (d *DHTServer) SendPING(node DHTNode) error {
	client, err := d.account.Client(node.Fingerprint)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s%s", node.Address(), DHT_PING_PATH)
	_, err = client.Get(url)

	return err
}

// RecievePING KAD (2.3) responds with http.StatusOK
func (d *DHTServer) RecievePING(w http.ResponseWriter, r *http.Request) {
	d.addNode(r)
}

func (d *DHTServer) SendFIND_NODE(fingerprint Fingerprint) (DHTNode, error) {
	client, err := d.account.Client(fingerprint)
	if err != nil {
		return DHTNode{}, err
	}

	var found *DHTNode
	ctx, cancel := context.WithCancel(context.Background())
	nodes := d.routingTable.nearest(fingerprint)
	// TODO make sure you don't duplicate nodes
	var lock sync.Mutex
	for i := 0; i < DHT_ALPHA; i++ {
		go func() {
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

					node := nodes[len(nodes)-1]
					nodes = nodes[:len(nodes)-1]
					lock.Unlock()

					url := fmt.Sprintf("%s%s", node.Address(), DHT_PING_PATH)
					resp, err := client.Get(url)
					if err != nil {
						continue
					}

					// Add it to the known nodes
					d.routingTable.addNode(node, d.SendPING)

					foundNodes := []DHTNode{}
					body, err := io.ReadAll(resp.Body)
					if err != nil {
						continue
					}

					err = json.Unmarshal(body, &foundNodes)
					if err != nil {
						continue
					}

					lock.Lock()
					for i := len(foundNodes) - 1; i >= 0; i-- {
						nodes = append(nodes, foundNodes[i])
						if foundNodes[i].Fingerprint == fingerprint {
							found = &foundNodes[i]
							cancel()
							break
						}
					}
					lock.Unlock()
				}
			}
		}()
	}

	cancel()
	if found == nil {
		return DHTNode{}, io.EOF
	}

	return *found, err
}

func (d *DHTServer) RecieveFIND_NODE(w http.ResponseWriter, r *http.Request) {
	d.addNode(r)

	fingerprintParam := r.FormValue("fingerprint")
	fingerprint, err := ParseFingerprint(fingerprintParam)
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

func (d *DHTServer) addNode(r *http.Request) {
	fingerprint, err := certToFingerprint(r.TLS.PeerCertificates)
	if err != nil {
		return
	}

	port, err := strconv.Atoi(r.Header.Get("Port"))
	if err != nil {
		return
	}

	node := DHTNode{
		Fingerprint: fingerprint,
		IP:          net.ParseIP(r.Header.Get("IP")),
		Port:        port,
	}

	d.routingTable.addNode(node, d.SendPING)
}

// Binary operations

// xor two fingerprints
func xor(a, b Fingerprint) (c Fingerprint) {
	for i := range a {
		c[i] = a[i] ^ b[i]
	}

	return
}

// prefixDiff counts the number of equal prefixed bits of a and b.
func prefixDiff(a, b Fingerprint, n int) int {
	buf, total := xor(a, b), 0

	for i, b := range buf {
		if 8*i >= n {
			break
		}

		if n > 8*i && n < 8*(i+1) {
			shift := 8 - uint(n%8)
			b >>= shift
		}

		total += bits.OnesCount8(b)
	}

	return total
}

// prefixLen returns the number of prefixed zero bits of a.
func prefixLen(a []byte) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// sortByDistance sorts ids by descending XOR distance with respect to id.
func sortByDistance(id Fingerprint, ids []DHTNode) []DHTNode {
	sort.Slice(ids, func(i, j int) bool {
		ixor := xor(ids[i].Fingerprint, id)
		jxor := xor(ids[j].Fingerprint, id)
		return bytes.Compare(ixor[:], jxor[:]) == -1
	})

	return ids
}
