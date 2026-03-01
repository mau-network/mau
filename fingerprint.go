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
		return fingerprintFromRSA(cert)
	case x509.Ed25519:
		return fingerprintFromEd25519(cert)
	case x509.ECDSA:
		// ECDSA support: skip for now as Mau primarily uses RSA/Ed25519
		return nil, nil
	default:
		return nil, x509.ErrUnsupportedAlgorithm
	}
}

func fingerprintFromRSA(cert *x509.Certificate) ([]byte, error) {
	if pubkey, ok := cert.PublicKey.(*rsa.PublicKey); ok {
		return packet.NewRSAPublicKey(cert.NotBefore, pubkey).Fingerprint, nil
	}
	return nil, nil
}

func fingerprintFromEd25519(cert *x509.Certificate) ([]byte, error) {
	// For Ed25519, extract fingerprint from DNSNames
	// (it's embedded there during certificate creation)
	for _, name := range cert.DNSNames {
		if len(name) == 40 { // SHA-1 fingerprint is 40 hex chars
			fpBytes, err := hex.DecodeString(name)
			if err == nil && len(fpBytes) == 20 {
				return fpBytes, nil
			}
		}
	}
	// Fallback: Should not reach here for Mau-generated Ed25519 certs
	return nil, errors.New("Ed25519 fingerprint not found in certificate DNSNames")
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
