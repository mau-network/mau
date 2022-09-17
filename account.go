package main

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"path"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
	keybasersa "github.com/keybase/go-crypto/rsa"
)

var (
	ErrPassphraseRequired   = errors.New("Passphrase must be specified")
	ErrIncorrectPassphrase  = errors.New("Incorrect passphrase")
	ErrNoIdentity           = errors.New("Can't find identity")
	ErrAccountAlreadyExists = errors.New("Account already exists")
)

func mauDir(rootPath string) string {
	return path.Join(rootPath, mauDirName)
}

func accountFile(rootPath string) string {
	return path.Join(mauDir(rootPath), accountKeyFilename)
}

func NewAccount(rootPath, name, email, passphrase string) (*Account, error) {
	if len(passphrase) == 0 {
		return nil, ErrPassphraseRequired
	}

	err := os.MkdirAll(mauDir(rootPath), dirPerm)
	if err != nil {
		return nil, err
	}

	if _, err := os.Stat(accountFile(rootPath)); err == nil {
		return nil, ErrAccountAlreadyExists
	}

	entity, err := openpgp.NewEntity(name, "", email, &packet.Config{
		DefaultCompressionAlgo: packet.CompressionZIP,
		RSABits:                rsaKeyLength,
	})
	if err != nil {
		return nil, err
	}

	plainFile, err := os.Create(accountFile(rootPath))
	if err != nil {
		return nil, err
	}

	encryptedFile, err := openpgp.SymmetricallyEncrypt(plainFile, []byte(passphrase), nil, nil)
	if err != nil {
		return nil, err
	}
	defer encryptedFile.Close()

	err = entity.SerializePrivate(encryptedFile, nil)
	if err != nil {
		return nil, err
	}

	account := Account{entity: entity, path: rootPath}
	return &account, nil
}

func OpenAccount(rootPath, passphrase string) (*Account, error) {
	err := os.MkdirAll(mauDir(rootPath), dirPerm)
	if err != nil {
		return nil, err
	}

	encryptedFile, err := os.Open(accountFile(rootPath))
	if err != nil {
		return nil, err
	}
	defer encryptedFile.Close()

	prompted := false
	prompt := func(keys []openpgp.Key, symmetric bool) ([]byte, error) {
		if prompted {
			return nil, ErrIncorrectPassphrase
		}

		prompted = true
		return []byte(passphrase), nil
	}

	decryptedFile, err := openpgp.ReadMessage(encryptedFile, openpgp.EntityList{}, prompt, nil)
	if err != nil {
		return nil, err
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(decryptedFile.UnverifiedBody))
	if err != nil {
		return nil, err
	}

	return &Account{entity: entity, path: rootPath}, nil
}

type Account struct {
	path   string
	entity *openpgp.Entity
}

func (a *Account) Identity() (string, error) {
	for _, i := range a.entity.Identities {
		return i.Name, nil
	}

	return "", ErrNoIdentity
}

func (a *Account) Name() string {
	for _, i := range a.entity.Identities {
		return i.UserId.Name
	}

	return ""
}

func (a *Account) Email() string {
	for _, i := range a.entity.Identities {
		return i.UserId.Email
	}

	return ""
}

func (a *Account) Fingerprint() Fingerprint {
	return a.entity.PrimaryKey.Fingerprint
}

func (a *Account) Export() ([]byte, error) {
	w := bytes.NewBuffer([]byte{})

	armored, err := armor.Encode(w, openpgp.PublicKeyType, map[string]string{})
	if err != nil {
		return nil, err
	}

	err = a.entity.Serialize(armored)
	armored.Close()
	if err != nil {
		return nil, nil
	}

	return w.Bytes(), nil
}

// TODO should we test this or make it internal. it's useful if someone want to
// provide an HTTP server implementation
func (a *Account) Certificate() (*tls.Certificate, error) {
	template := x509.Certificate{
		Version:      3,
		SerialNumber: big.NewInt(1),
		NotBefore:    a.entity.PrimaryKey.CreationTime,
		NotAfter:     a.entity.PrimaryKey.CreationTime.AddDate(100, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}

	privkey, ok := a.entity.PrivateKey.PrivateKey.(*keybasersa.PrivateKey)
	if !ok {
		return nil, errors.New("Can't convert private key")
	}

	pubkey, ok := a.entity.PrimaryKey.PublicKey.(*keybasersa.PublicKey)
	if !ok {
		return nil, errors.New("Can't convert public key")
	}

	crtvalues := []rsa.CRTValue{}
	for _, i := range privkey.Precomputed.CRTValues {
		crtvalues = append(crtvalues, rsa.CRTValue(i))
	}

	rsakey := rsa.PrivateKey{
		PublicKey: rsa.PublicKey{
			N: pubkey.N,
			E: int(pubkey.E),
		},
		D:      privkey.D,
		Primes: privkey.Primes,
		Precomputed: rsa.PrecomputedValues{
			Dp:        privkey.Precomputed.Dp,
			Dq:        privkey.Precomputed.Dq,
			Qinv:      privkey.Precomputed.Qinv,
			CRTValues: crtvalues,
		},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &rsakey.PublicKey, &rsakey)
	if err != nil {
		return nil, err
	}

	keyPem := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(&rsakey)}
	certPem := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}

	keyPemBytes := bytes.NewBuffer([]byte{})
	pem.Encode(keyPemBytes, keyPem)

	certPemBytes := bytes.NewBuffer([]byte{})
	pem.Encode(certPemBytes, certPem)

	cert, err := tls.X509KeyPair(certPemBytes.Bytes(), keyPemBytes.Bytes())
	if err != nil {
		return nil, err
	}

	return &cert, nil
}
