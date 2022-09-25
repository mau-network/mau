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

func TestXor(t *testing.T) {
	fpr1, _ := ParseFingerprint("0000000000000000000000000000000000000F0F")
	fpr2, _ := ParseFingerprint("00000000000000000000000000000000000000FF")

	res, _ := ParseFingerprint("0000000000000000000000000000000000000FF0")
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
	v, _ := ParseFingerprint(fpr)
	return v
}
