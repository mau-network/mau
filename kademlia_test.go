package mau

import (
	"testing"
)

func TestBucket(t *testing.T) {
	b := bucket{}

	ASSERT_EQUAL(t, false, b.isFull())
	ASSERT_EQUAL(t, nil, b.get(Fingerprint{}))

	fpr1, err := ParseFingerprint("ABAF11C65A2970B130ABE3C479BE3E4300411886")
	ASSERT_NO_ERROR(t, err)

	fpr2, err := ParseFingerprint("AAAF11C65A2970B130ABE3C479BE3E4300411886")
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

func TestNewDHTRPC(t *testing.T) {
	account, err := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "password")
	ASSERT_NO_ERROR(t, err)

	listener, _ := TempListener()
	server, err := account.Server(nil)
	go server.Serve(*listener)

	REFUTE_EQUAL(t, nil, server)
}
