package mau

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"

	
	"github.com/ProtonMail/go-crypto/openpgp/packet"
)

var (
	ErrCantFindFingerprint = errors.New("Can't find fingerprint.")
	ErrCantFindAddress     = errors.New("Can't find address (DNSName) in certificate.")
)

// Fingerprint represents an OpenPGP key fingerprint.
// Length varies by key version:
//   - v4 keys: 20 bytes (SHA-1 hash)
//   - v5/v6 keys: 32 bytes (SHA-256 hash)
type Fingerprint []byte

func (f Fingerprint) String() string {
	return hex.EncodeToString(f)
}

// Equal compares two fingerprints for equality.
func (f Fingerprint) Equal(other Fingerprint) bool {
	return bytes.Equal(f, other)
}

func (f Fingerprint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, f)), nil
}

func (f *Fingerprint) UnmarshalJSON(b []byte) error {
	if bytes.Equal(b, []byte("null")) {
		return nil
	}

	// Remove quotes
	if len(b) < 2 {
		return fmt.Errorf("invalid fingerprint JSON: too short")
	}

	v := string(b[1 : len(b)-1])
	pf, err := FingerprintFromString(v)
	if err != nil {
		return err
	}

	*f = pf
	return nil
}

func FingerprintFromString(s string) (Fingerprint, error) {
	fprParsed, err := hex.DecodeString(s)
	if err != nil {
		return nil, err
	}
	return Fingerprint(fprParsed), nil
}

func fingerprintFromPublicKey(cert *x509.Certificate) ([]byte, error) {
	switch cert.PublicKeyAlgorithm {
	case x509.RSA:
		if pubkey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
			return packet.NewRSAPublicKey(cert.NotBefore, pubkey).Fingerprint, nil
		}
	case x509.ECDSA:
		// TODO: ECDSA support needs investigation - ProtonMail fork may have different signature
		// For now, skip ECDSA as Mau primarily uses RSA 4096
		return nil, nil
	default:
		return nil, x509.ErrUnsupportedAlgorithm
	}
	return nil, nil
}

func FingerprintFromCert(certs []*x509.Certificate) (Fingerprint, error) {
	for _, cert := range certs {
		fpSlice, err := fingerprintFromPublicKey(cert)
		if err != nil {
			return nil, err
		}
		if fpSlice != nil {
			return Fingerprint(fpSlice), nil
		}
	}
	return nil, ErrCantFindFingerprint
}

func certToAddress(certs []*x509.Certificate) (string, error) {
	for _, cert := range certs {
		for _, name := range cert.DNSNames {
			return name, nil
		}
	}

	return "", ErrCantFindAddress
}
