package mau

import (
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/openpgp/packet"
)

var ErrIncorrectFingerprintLength = errors.New("Provided fingerprint length is not correct")

type Fingerprint [20]byte

func (f Fingerprint) String() string {
	return hex.EncodeToString(f[:])
}

func ParseFingerprint(s string) (fpr Fingerprint, err error) {
	decodeLen := hex.DecodedLen(len(s))
	if decodeLen != cap(fpr) {
		err = fmt.Errorf("%w : provided: %d", ErrIncorrectFingerprintLength, decodeLen)
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

func certIncludes(rawCerts [][]byte, fingerprint Fingerprint) error {
	for _, rawcert := range rawCerts {
		certs, err := x509.ParseCertificates(rawcert)
		if err != nil {
			return err
		}

		for _, cert := range certs {
			switch cert.PublicKeyAlgorithm {
			case x509.RSA:
				if pubkey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
					var id Fingerprint = packet.NewRSAPublicKey(cert.NotBefore, pubkey).Fingerprint
					if fingerprint == id {
						return nil
					}
				}
			case x509.ECDSA:
				if pubkey, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
					var id Fingerprint = packet.NewECDSAPublicKey(cert.NotBefore, pubkey).Fingerprint
					if fingerprint == id {
						return nil
					}
				}
			default:
				return x509.ErrUnsupportedAlgorithm
			}
		}
	}

	return ErrIncorrectPeerCertificate
}
