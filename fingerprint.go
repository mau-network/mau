package main

import (
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrIncorrectFingerprinLength = errors.New("Provided fingerprint length is not correct")

type Fingerprint [20]byte

func (f Fingerprint) String() string {
	return hex.EncodeToString(f[:])
}

func ParseFingerprint(s string) (fpr Fingerprint, err error) {
	decodeLen := hex.DecodedLen(len(s))
	if decodeLen != cap(fpr) {
		err = fmt.Errorf("%w : provided: %d", ErrIncorrectFingerprinLength, decodeLen)
		return
	}

	var fprParsed []byte
	fprParsed, err = hex.DecodeString(s)
	if err != nil {
		return
	}

	copy(fpr[:], fprParsed)
	return
}
