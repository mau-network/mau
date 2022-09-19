package mau

import (
	"bytes"
	"encoding/json"
	"errors"
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

var (
	ErrPeerNodeCertIncorrect = errors.New("Peer Certificate didn't match expected Identity.")
)

type DHTNode struct {
	Key     Fingerprint
	Address string
}
type Bucket struct {
	Nodes []DHTNode
}

type DHTRPC struct {
	StoreStorage map[Fingerprint]*DHTNode
}

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
// TODO When sending requests make sure we're connecting to the correct user by
// checking the TLS peer cert
func (d *DHTRPC) SendPING(node *DHTNode) bool {
	resp, err := http.Get(node.Address + DHT_PING_PATH)
	return err == nil &&
		resp.StatusCode == http.StatusOK
}

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) RecievePING(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) SendSTORE(node *DHTNode, value *DHTNode) error {
	body, err := json.Marshal(value)
	if err != nil {
		return err
	}

	resp, err := http.Post(node.Address+DHT_STORE_PATH, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Response from peer: %s", resp.Status)
	}

	return nil
}

// Kademlia: A Peer-to-Peer Information System Based on the XOR Metric (2.3)
func (d *DHTRPC) RecieveSTORE(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var node DHTNode
	if err := json.Unmarshal(body, &node); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Whoever asks us for STORE needs to have the identity in the value
	// if !IsNodeCert(node.Key, r.TLS) {
	// 	w.WriteHeader(http.StatusUnauthorized)
	// 	return
	// }

	d.StoreStorage[node.Key] = &node
}

func (d *DHTRPC) SendFIND_NODE(node *DHTNode, key Fingerprint) ([]DHTNode, error) {
	resp, err := http.Post(node.Address+DHT_STORE_PATH, "application/octet-stream", bytes.NewBuffer(key[:]))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Response from peer: %s", resp.Status)
	}

	var nodes []DHTNode
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (d *DHTRPC) ReciveFIND_NODE(w http.ResponseWriter, r *http.Request) {
	// key, err := io.ReadAll(r.Body)
	// defer r.Body.Close()
	// if err != nil {

	// }
}

func (d *DHTRPC) SendFIND_VALUE(node *DHTNode) {
}

func (d *DHTRPC) RecieveFIND_VALUE(w http.ResponseWriter, r *http.Request) {
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
		return bytes.Compare(XOR(ids[i].Key[:], id[:]), XOR(ids[j].Key[:], id[:])) == -1
	})

	return ids
}
