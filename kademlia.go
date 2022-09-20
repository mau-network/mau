package mau

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/bits"
	"net/http"
	"sort"
)

const (
	DHT_K               = 20 // number of buckets
	DHT_PING_PATH       = "/kad/ping"
	DHT_STORE_PATH      = "/kad/store"
	DHT_FIND_NODE_PATH  = "/kad/find_node"
	DHT_FIND_VALUE_PATH = "/kad/find_value"
)

type DHTNode struct {
	Fingerprint Fingerprint
	Address     string
}

type bucket struct {
	nodes []DHTNode
}

type DHTRPC struct {
	mux          *http.ServeMux
	account      *Account
	buckets      [DHT_K]bucket
	storeStorage map[Fingerprint]*DHTNode
}

func NewDHTRPC(account *Account) *DHTRPC {
	d := &DHTRPC{
		mux:          http.NewServeMux(),
		account:      account,
		storeStorage: map[Fingerprint]*DHTNode{},
	}

	d.mux.HandleFunc(DHT_PING_PATH, d.RecievePING)
	d.mux.HandleFunc(DHT_STORE_PATH, d.RecieveSTORE)
	d.mux.HandleFunc(DHT_FIND_NODE_PATH, d.ReciveFIND_NODE)
	d.mux.HandleFunc(DHT_FIND_VALUE_PATH, d.RecieveFIND_VALUE)

	return d
}

func (d *DHTRPC) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.mux.ServeHTTP(w, r)
}

// SendPING sends a ping to a node and returns true if the node response status isn't 2xx
//
// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) SendPING(node *DHTNode, client *Client) bool {
	_, err := client.Get(node.Address + DHT_PING_PATH)
	return err == nil
}

// RecievePING responds with http.StatusOK
//
// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) RecievePING(w http.ResponseWriter, r *http.Request) {}

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) SendSTORE(node *DHTNode, value *DHTNode, client *Client) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	resp, err := client.Post(node.Address+DHT_STORE_PATH, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Response from peer: %s", resp.Status)
	}

	return nil
}

// RecieveSTORE stories the body of the request node for later Find_VALUE call
// Instead of asking the client for the identity this call gets if from the TLS certificate
//
// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) RecieveSTORE(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fingerprint, err := certToFingerprint(r.TLS.PeerCertificates)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	d.storeStorage[fingerprint] = &DHTNode{
		Fingerprint: fingerprint,
		Address:     string(body),
	}
}

func (d *DHTRPC) ReciveFIND_NODE(w http.ResponseWriter, r *http.Request) {
	// TODO
	// key, err := io.ReadAll(r.Body)
	// defer r.Body.Close()
	// if err != nil {

	// }
}

func (d *DHTRPC) SendFIND_VALUE(node *DHTNode) {
	// TODO
}

func (d *DHTRPC) RecieveFIND_VALUE(w http.ResponseWriter, r *http.Request) {
	// TODO
}

// Binary operations

// XOR allocates a new byte slice with the computed result of XOR(a, b).
func XOR(a, b []byte) []byte {
	if len(a) != len(b) {
		return a
	}

	c := make([]byte, len(a))

	for i := 0; i < len(a); i++ {
		c[i] = a[i] ^ b[i]
	}

	return c
}

// PrefixDiff counts the number of equal prefixed bits of a and b.
func PrefixDiff(a, b []byte, n int) int {
	buf, total := XOR(a, b), 0

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

// PrefixLen returns the number of prefixed zero bits of a.
func PrefixLen(a []byte) int {
	for i, b := range a {
		if b != 0 {
			return i*8 + bits.LeadingZeros8(b)
		}
	}

	return len(a) * 8
}

// SortByDistance sorts ids by descending XOR distance with respect to id.
func SortByDistance(id Fingerprint, ids []DHTNode) []DHTNode {
	sort.Slice(ids, func(i, j int) bool {
		return bytes.Compare(XOR(ids[i].Fingerprint[:], id[:]), XOR(ids[j].Fingerprint[:], id[:])) == -1
	})

	return ids
}
