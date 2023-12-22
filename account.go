package mau

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"sort"
	"time"

	_ "crypto/sha256"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
)

var (
	ErrPassphraseRequired      = errors.New("Passphrase must be specified")
	ErrIncorrectPassphrase     = errors.New("Incorrect passphrase")
	ErrNoIdentity              = errors.New("Can't find identity")
	ErrAccountAlreadyExists    = errors.New("Account already exists")
	ErrCannotConvertPrivateKey = errors.New("Can't convert private key")
	ErrCannotConvertPublicKey  = errors.New("Can't convert public key")
)

func mauDir(d string) string      { return path.Join(d, mauDirName) }
func accountFile(d string) string { return path.Join(mauDir(d), accountKeyFilename) }

func NewAccount(root, name, email, passphrase string) (*Account, error) {
	if len(passphrase) == 0 {
		return nil, ErrPassphraseRequired
	}

	dir := mauDir(root)

	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, err
	}

	acc := accountFile(root)
	if _, err := os.Stat(acc); err == nil {
		return nil, ErrAccountAlreadyExists
	}

	entity, err := openpgp.NewEntity(name, "", email, &packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCompressionAlgo: packet.CompressionZIP,
		RSABits:                rsaKeyLength,
	})
	if err != nil {
		return nil, err
	}

	plainFile, err := os.Create(acc)
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

	return &Account{
		entity: entity,
		path:   root,
	}, nil
}

func OpenAccount(rootPath, passphrase string) (*Account, error) {
	dir := mauDir(rootPath)
	err := os.MkdirAll(dir, dirPerm)
	if err != nil {
		return nil, err
	}

	acc := accountFile(rootPath)
	encryptedFile, err := os.Open(acc)
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

	return &Account{
		entity: entity,
		path:   rootPath,
	}, nil
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

func (a *Account) Export(w io.Writer) error {
	armored, err := armor.Encode(w, openpgp.PublicKeyType, map[string]string{})
	if err != nil {
		return err
	}

	err = a.entity.Serialize(armored)
	armored.Close()
	if err != nil {
		return nil
	}

	return nil
}

func (a *Account) certificate(DNSNames []string) (cert tls.Certificate, err error) {
	template := x509.Certificate{
		Version:      3,
		DNSNames:     DNSNames,
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

	privkey, ok := a.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		err = ErrCannotConvertPrivateKey
		return
	}

	pubkey, ok := a.entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
	if !ok {
		err = ErrCannotConvertPublicKey
		return
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

	var derBytes []byte
	derBytes, err = x509.CreateCertificate(rand.Reader, &template, &template, &rsakey.PublicKey, &rsakey)
	if err != nil {
		return
	}

	keyPem := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(&rsakey)}
	certPem := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}

	var keyPemBytes bytes.Buffer
	pem.Encode(&keyPemBytes, keyPem)

	var certPemBytes bytes.Buffer
	pem.Encode(&certPemBytes, certPem)

	return tls.X509KeyPair(certPemBytes.Bytes(), keyPemBytes.Bytes())
}

func (a *Account) AddFile(r io.Reader, name string, recipients []*Friend) (*File, error) {
	if path.Ext(name) != ".pgp" {
		name += ".pgp"
	}

	fpr := a.Fingerprint().String()

	if err := os.MkdirAll(path.Join(a.path, fpr), dirPerm); err != nil {
		return nil, err
	}

	p := path.Join(a.path, fpr, name)
	if _, err := os.Stat(p); err == nil {
		file := File{Path: p}
		np, err := file.Hash()
		if err != nil {
			return nil, err
		}

		if err = os.MkdirAll(p+".versions", dirPerm); err != nil {
			return nil, err
		}

		if err = os.Rename(p, path.Join(p+".versions", np)); err != nil {
			return nil, err
		}
	}

	entities := []*openpgp.Entity{a.entity}
	for _, f := range recipients {
		entities = append(entities, f.entity)
	}

	file, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	w, err := openpgp.Encrypt(file, entities, a.entity, nil, nil)
	if err != nil {
		return nil, err
	}

	io.Copy(w, r)
	w.Close()

	return &File{Path: p}, nil
}

func (a *Account) RemoveFile(file *File) error {
	err := os.Remove(file.Path)
	if err != nil {
		return err
	}

	if file.version {
		return nil
	}

	for _, version := range file.Versions() {
		err := a.RemoveFile(version)
		if err != nil {
			return err
		}
	}

	return nil
}

func (a *Account) ListFiles(fingerprint Fingerprint, after time.Time, limit uint) []*File {
	followedPath := path.Join(a.path, fingerprint.String())
	unfollowedPath := path.Join(a.path, "."+fingerprint.String())
	var dirpath string

	if _, err := os.Stat(followedPath); err == nil {
		dirpath = followedPath
	} else if _, err := os.Stat(unfollowedPath); err == nil {
		dirpath = unfollowedPath
	} else {
		return []*File{}
	}

	files, err := os.ReadDir(dirpath)
	if err != nil {
		return []*File{}
	}

	type dirEntry struct {
		entry        fs.DirEntry
		info         fs.FileInfo
		modification time.Time
	}

	recent := []dirEntry{}
	for _, f := range files {
		if !f.Type().IsRegular() {
			continue
		}

		info, err := f.Info()
		if err != nil {
			continue
		}

		mod := info.ModTime()
		if mod.Before(after) {
			continue
		}

		recent = append(recent, dirEntry{
			entry:        f,
			modification: mod,
		})
	}

	sort.Slice(recent, func(i, j int) bool {
		return recent[i].modification.Before(recent[j].modification)
	})

	if uint(len(recent)) < limit {
		limit = uint(len(recent))
	}

	page := recent

	// if limit was 0 then it's unlimited
	if limit > 0 {
		page = recent[:limit]
	}

	list := make([]*File, 0, len(page))
	for _, item := range page {
		list = append(list, &File{
			Path: path.Join(dirpath, item.entry.Name()),
		})
	}

	return list
}
