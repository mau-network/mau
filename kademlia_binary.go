package mau

import (
	"bytes"
	"math/bits"
	"sort"
)

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
	sort.Slice(peers, func(i, j int) bool {
		ixor := xor(peers[i].Fingerprint, fingerprint)
		jxor := xor(peers[j].Fingerprint, fingerprint)
		return bytes.Compare(ixor[:], jxor[:]) == -1
	})
}
