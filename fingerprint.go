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
	if bytes.Equal(b, []byte("null")) {
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

	*f = pf

	return nil
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
			return Fingerprint{}, err
		}
		if fpSlice != nil {
			var fp Fingerprint
			copy(fp[:], fpSlice)
			return fp, nil
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
