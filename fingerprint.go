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
var ErrCantFindFingerprint = errors.New("Can't find fingerprint.")
var ErrCantFindAddress = errors.New("Can't find address (DNSName) in certificate.")

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

func certToFingerprint(certs []*x509.Certificate) (Fingerprint, error) {
	for _, cert := range certs {
		switch cert.PublicKeyAlgorithm {
		case x509.RSA:
			if pubkey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
				return packet.NewRSAPublicKey(cert.NotBefore, pubkey).Fingerprint, nil

			}
		case x509.ECDSA:
			if pubkey, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
				return packet.NewECDSAPublicKey(cert.NotBefore, pubkey).Fingerprint, nil
			}
		default:
			return Fingerprint{}, x509.ErrUnsupportedAlgorithm
		}
	}

	return Fingerprint{}, ErrCantFindFingerprint
}

func certToAddress(certs []*x509.Certificate) (string, error) {
	for _, cert := range certs {
		for _, name := range cert.DNSNames {
			return name, nil
		}
	}

	return "", ErrCantFindAddress
}
