package mau

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestBucket(t *testing.T) {
	b := bucket{}

	ASSERT_EQUAL(t, false, b.isFull())
	ASSERT_EQUAL(t, nil, b.get(Fingerprint{}))

	fpr1, err := FingerprintFromString("ABAF11C65A2970B130ABE3C479BE3E4300411886")
	ASSERT_NO_ERROR(t, err)

	fpr2, err := FingerprintFromString("AAAF11C65A2970B130ABE3C479BE3E4300411886")
	ASSERT_NO_ERROR(t, err)

	b.addToTail(&Peer{fpr1, "address"})
	ASSERT_EQUAL(t, fpr1, b.get(fpr1).Fingerprint)
	ASSERT_EQUAL(t, fpr1, b.leastRecentlySeen().Fingerprint)

	b.addToTail(&Peer{fpr2, "address2"})
	ASSERT_EQUAL(t, fpr1, b.leastRecentlySeen().Fingerprint)

	b.moveToTail(b.get(fpr1))
	ASSERT_EQUAL(t, fpr2, b.leastRecentlySeen().Fingerprint)

	b.moveToTail(b.get(fpr2))
	ASSERT_EQUAL(t, fpr1, b.leastRecentlySeen().Fingerprint)

	ASSERT_EQUAL(t, 2, len(b.dup()))

	rando := b.randomPeer()
	ASSERT(
		t,
		rando.Fingerprint == fpr1 || rando.Fingerprint == fpr2,
		"rando should have returned one of the fingerprints instead: %s",
		rando.Fingerprint,
	)

	b.remove(b.get(fpr1))
	ASSERT_EQUAL(t, fpr2, b.leastRecentlySeen().Fingerprint)
}

func TestNewDHTServer(t *testing.T) {
	account, _ := NewAccount(t.TempDir(), "Main peer", "main@example.com", "password")
	s := newDHTServer(account, "localhost:80")

	REFUTE_EQUAL(t, nil, s.mux)
	ASSERT_EQUAL(t, account, s.account)
	ASSERT_EQUAL(t, "localhost:80", s.address)
}

func TestReceivePing(t *testing.T) {
	account, _ := NewAccount(t.TempDir(), "Main peer", "main@example.com", "password")
	s := newDHTServer(account, "localhost:80")

	t.Run("without mTLS", func(t *testing.T) {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/kad/ping", &bytes.Buffer{})
		s.receivePing(w, r)

		ASSERT_EQUAL(t, http.StatusBadRequest, w.Result().StatusCode)
	})

	t.Run("with mTLS", func(t *testing.T) {
		listener, bootstrap_addr := TempListener()
		server, err := account.Server(nil)
		go server.Serve(*listener, bootstrap_addr)
		defer server.Close()
		for ; server.dhtServer == nil; time.Sleep(time.Millisecond) {
		}

		peer, _ := NewAccount(t.TempDir(), "Peer", "peer@example.com", "password")
		ASSERT_NO_ERROR(t, err)

		client, err := peer.Client(account.Fingerprint(), []string{"localhost:90"})
		ASSERT_NO_ERROR(t, err)

		u := url.URL{
			Scheme: uriProtocolName,
			Host:   bootstrap_addr,
			Path:   "/kad/ping",
		}
		resp, err := client.client.R().Get(u.String())
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode())
	})
}

func TestDHTServerAddPeer(t *testing.T) {
	account, _ := NewAccount(t.TempDir(), "Main peer", "main@example.com", "password")
	s := newDHTServer(account, "localhost:80")

	peerFpr, err := FingerprintFromString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF")
	ASSERT_NO_ERROR(t, err)

	peerFpr2, err := FingerprintFromString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFE")
	ASSERT_NO_ERROR(t, err)

	distance := s.bucketFor(peerFpr)
	ASSERT_EQUAL(t, 0, len(s.buckets[distance].values))

	s.addPeer(&Peer{peerFpr, "address"})
	ASSERT_EQUAL(t, 1, len(s.buckets[distance].values))

	s.addPeer(&Peer{peerFpr2, "address"})
	ASSERT_EQUAL(t, 2, len(s.buckets[distance].values))
}

func TestDHTServer(t *testing.T) {
	bootstrap, err := NewAccount(t.TempDir(), "Main peer", "main@example.com", "password")
	listener, bootstrap_addr := TempListener()
	bootstrap_peer := &Peer{bootstrap.Fingerprint(), bootstrap_addr}

	server, err := bootstrap.Server(nil)
	ASSERT_NO_ERROR(t, err)
	REFUTE_EQUAL(t, nil, server)
	go server.Serve(*listener, bootstrap_addr)
	defer server.Close()

	peers := []*Account{}
	servers := []*Server{}
	const COUNT = 10

	t.Run("Servers initialization", func(t *testing.T) {
		for i := 0; i < COUNT; i++ {
			a, err := NewAccount(t.TempDir(), fmt.Sprintf("Peer %d", i), fmt.Sprintf("peer%d@example.com", i), "password")
			ASSERT_NO_ERROR(t, err)
			peers = append(peers, a)

			s, err := a.Server([]*Peer{bootstrap_peer})
			ASSERT_NO_ERROR(t, err)
			REFUTE_EQUAL(t, nil, s)
			servers = append(servers, s)

			l, addr := TempListener()
			go s.Serve(*l, addr)
		}
	})

	t.Run("Lookup bootstrap peer and ping it", func(t *testing.T) {
		for _, s := range servers {
			for ; s.dhtServer == nil; time.Sleep(time.Millisecond) {
			}
			b := s.dhtServer.sendFindPeer(context.Background(), bootstrap.Fingerprint())
			ASSERT_EQUAL(t, bootstrap.Fingerprint(), b.Fingerprint)
			err := s.dhtServer.sendPing(b)
			ASSERT_NO_ERROR(t, err)
		}
	})

	t.Run("Bootstrap contact list", func(t *testing.T) {
		c, _ := bootstrap.Client(bootstrap_peer.Fingerprint, []string{bootstrap_addr})
		u := url.URL{
			Scheme: uriProtocolName,
			Path:   dht_FIND_PEER_PATH + bootstrap.Fingerprint().String(),
			Host:   bootstrap_addr,
		}

		var peers []Peer
		_, err := c.client.
			R().
			ForceContentType("application/json").
			SetResult(&peers).
			Get(u.String())

		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, COUNT, len(peers))
	})

	t.Run("Lookup each other", func(t *testing.T) {
		for _, s := range servers {
			for _, p := range peers {
				if p.Fingerprint() == s.account.Fingerprint() {
					continue
				}

				b := s.dhtServer.sendFindPeer(context.Background(), p.Fingerprint())
				REFUTE_EQUAL(t, nil, b)
				ASSERT_EQUAL(t, p.Fingerprint(), b.Fingerprint)
			}
		}
	})

	t.Run("looking up unknown peer", func(t *testing.T) {
		for _, s := range servers {
			s.dhtServer.refreshAllBuckets(context.Background())
			c, _ := bootstrap.Client(s.account.Fingerprint(), []string{bootstrap_addr})
			u := url.URL{
				Scheme: uriProtocolName,
				Path:   dht_FIND_PEER_PATH + "0000000000000000000000000000000000000F0F",
				Host:   s.dhtServer.address,
			}

			var peers []Peer
			_, err := c.client.R().
				ForceContentType("application/json").
				SetResult(&peers).
				Get(u.String())

			ASSERT_NO_ERROR(t, err)

			// (Outdated note for dev) review the logic of this part, the previous test asserts that peers know each other but now it returns only one peer, what's going on here?

			// (comment on the comment) now I understand why this is going on
			// when asking about a peer in the previous sub test the node doesn't know about it
			// so it asks the bootstrap node and it returns the list of nodes it knows about, our node now can find the target and return it
			// without any more requests this is why the contact list include only the bootstrap node.
			// I understood that after hours of thinking and reading the paper. even though I implemented the fuckin thing it took me time.
			// maybe I should review the idea of implementing kad
			ASSERT_EQUAL(t, 1, len(peers))
		}
	})

	t.Run("Doesn't find an unknown fingerprint", func(t *testing.T) {
		for _, s := range servers {
			b := s.dhtServer.sendFindPeer(context.Background(), ParseFPRIgnoreErr("0000000000000000000000000000000000000F0F"))
			ASSERT_EQUAL(t, nil, b)
			break
		}
	})
}

func TestXor(t *testing.T) {
	fpr1, _ := FingerprintFromString("0000000000000000000000000000000000000F0F")
	fpr2, _ := FingerprintFromString("00000000000000000000000000000000000000FF")

	res, _ := FingerprintFromString("0000000000000000000000000000000000000FF0")
	ASSERT_EQUAL(t, res, xor(fpr1, fpr2))
}

func TestPrefixLen(t *testing.T) {
	tcs := []struct {
		fpr Fingerprint
		len int
	}{
		{ParseFPRIgnoreErr("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 0},
		{ParseFPRIgnoreErr("8FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 0},
		{ParseFPRIgnoreErr("7FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 1},
		{ParseFPRIgnoreErr("3FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 2},
		{ParseFPRIgnoreErr("1FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 3},
		{ParseFPRIgnoreErr("0FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 4},
		{ParseFPRIgnoreErr("07FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 5},
		{ParseFPRIgnoreErr("03FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 6},
		{ParseFPRIgnoreErr("01FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 7},
		{ParseFPRIgnoreErr("00FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 8},
		{ParseFPRIgnoreErr("007FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 9},
		{ParseFPRIgnoreErr("003FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFF"), 10},
		{ParseFPRIgnoreErr("000000000000000000000000000000000000000F"), 156},
		{ParseFPRIgnoreErr("0000000000000000000000000000000000000007"), 157},
		{ParseFPRIgnoreErr("0000000000000000000000000000000000000003"), 158},
		{ParseFPRIgnoreErr("0000000000000000000000000000000000000002"), 158},
		{ParseFPRIgnoreErr("0000000000000000000000000000000000000001"), 159},
		{ParseFPRIgnoreErr("0000000000000000000000000000000000000000"), 160},
	}

	for _, tc := range tcs {
		t.Run(tc.fpr.String(), func(t *testing.T) {
			ASSERT_EQUAL(t, tc.len, prefixLen(tc.fpr))
		})
	}
}

func ParseFPRIgnoreErr(fpr string) Fingerprint {
	v, _ := FingerprintFromString(fpr)
	return v
}
