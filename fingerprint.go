package mau

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	"golang.org/x/crypto/openpgp/packet"
)

var (
	ErrIncorrectFingerprintLength = errors.New("Provided fingerprint length is not correct")
	ErrCantFindFingerprint        = errors.New("Can't find fingerprint.")
	ErrCantFindAddress            = errors.New("Can't find address (DNSName) in certificate.")
)

const FINGERPRINT_LEN = 20

type Fingerprint [FINGERPRINT_LEN]byte

func (f Fingerprint) String() string {
	return hex.EncodeToString(f[:])
}

func (f Fingerprint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, f)), nil
}

func (f *Fingerprint) UnmarshalJSON(b []byte) error {
	if bytes.Compare(b, []byte("null")) == 0 {
		return nil
	}

	if len(b) != (FINGERPRINT_LEN*2)+2 {
		return fmt.Errorf("%w %s", ErrIncorrectFingerprintLength, b)
	}

	v := string(b[1 : len(b)-1])
	pf, err := FingerprintFromString(v)
	if err != nil {
		return err
	}

	for i := range pf {
		f[i] = pf[i]
	}

	return nil
}

func (fpr *Fingerprint) isInCert(rawCerts [][]byte) error {
	for _, rawcert := range rawCerts {
		certs, err := x509.ParseCertificates(rawcert)
		if err != nil {
			return err
		}

		// Go over all certs. check public key
		// if one of the keys fingerprint == fingerprint we return nil
		for _, cert := range certs {
			switch cert.PublicKeyAlgorithm {
			case x509.RSA:
				if pubkey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
					var id Fingerprint = packet.NewRSAPublicKey(cert.NotBefore, pubkey).Fingerprint
					if *fpr == id {
						return nil
					}
				}
			case x509.ECDSA:
				if pubkey, ok := cert.PublicKey.(*ecdsa.PublicKey); ok {
					var id Fingerprint = packet.NewECDSAPublicKey(cert.NotBefore, pubkey).Fingerprint
					if *fpr == id {
						return nil
					}
				}
			default:
				return x509.ErrUnsupportedAlgorithm
			}
		}
	}

	// non of the certs include fingerprint
	return ErrIncorrectPeerCertificate
}

func FingerprintFromString(s string) (fpr Fingerprint, err error) {
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

func FingerprintFromCert(certs []*x509.Certificate) (Fingerprint, error) {
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
