package mau

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"io"
	"io/fs"
	"math/big"
	"os"
	"path"
	"slices"
	"time"

	_ "crypto/sha256"

	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp"
	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp/armor"
	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
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

	err := os.MkdirAll(dir, DirPerm)
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
	err := os.MkdirAll(dir, DirPerm)
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
	if decryptedFile == nil {
		return nil, errors.New("openpgp.ReadMessage returned nil")
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
	if a == nil || a.entity == nil {
		return "", ErrNoIdentity
	}
	for _, i := range a.entity.Identities {
		return i.Name, nil
	}

	return "", ErrNoIdentity
}

func (a *Account) Name() string {
	if a == nil || a.entity == nil {
		return ""
	}
	
	// First, look for primary identity
	for _, i := range a.entity.Identities {
		if i.SelfSignature.IsPrimaryId != nil && *i.SelfSignature.IsPrimaryId {
			return i.UserId.Name
		}
	}
	
	// Fallback to first identity
	for _, i := range a.entity.Identities {
		return i.UserId.Name
	}

	return ""
}

func (a *Account) Email() string {
	if a == nil || a.entity == nil {
		return ""
	}
	
	// First, look for primary identity
	for _, i := range a.entity.Identities {
		if i.SelfSignature.IsPrimaryId != nil && *i.SelfSignature.IsPrimaryId {
			return i.UserId.Email
		}
	}
	
	// Fallback to first identity
	for _, i := range a.entity.Identities {
		return i.UserId.Email
	}

	return ""
}

func (a *Account) Fingerprint() Fingerprint {
	if a == nil || a.entity == nil || a.entity.PrimaryKey == nil {
		return Fingerprint{}
	}
	return a.entity.PrimaryKey.Fingerprint
}

func (a *Account) Export(w io.Writer) error {
	if a == nil || a.entity == nil {
		return errors.New("account or entity is nil")
	}
	armored, err := armor.Encode(w, openpgp.PublicKeyType, map[string]string{})
	if err != nil {
		return err
	}

	err = a.entity.Serialize(armored)
	armored.Close()
	if err != nil {
		return err
	}

	return nil
}

func (a *Account) certificate(DNSNames []string) (cert tls.Certificate, err error) {
	if a == nil || a.entity == nil || a.entity.PrimaryKey == nil || a.entity.PrivateKey == nil {
		err = errors.New("account or entity is incomplete")
		return
	}

	template := buildCertificateTemplate(DNSNames, a.entity.PrimaryKey.CreationTime)
	rsakey, err := extractRSAKeyFromEntity(a.entity)
	if err != nil {
		return tls.Certificate{}, err
	}

	derBytes, err := x509.CreateCertificate(nil, &template, &template, &rsakey.PublicKey, rsakey)
	if err != nil {
		return tls.Certificate{}, err
	}

	return encodeCertificateAndKey(rsakey, derBytes)
}

func buildCertificateTemplate(dnsNames []string, creationTime time.Time) x509.Certificate {
	return x509.Certificate{
		Version:      3,
		DNSNames:     dnsNames,
		SerialNumber: big.NewInt(1),
		NotBefore:    creationTime,
		NotAfter:     creationTime.AddDate(100, 0, 0),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}
}

func extractRSAKeyFromEntity(entity *openpgp.Entity) (*rsa.PrivateKey, error) {
	privkey, ok := entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrCannotConvertPrivateKey
	}

	pubkey, ok := entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, ErrCannotConvertPublicKey
	}

	crtvalues := []rsa.CRTValue{}
	//nolint:staticcheck // SA1019: CRTValues deprecated but needed for backward compatibility
	for _, i := range privkey.Precomputed.CRTValues {
		crtvalues = append(crtvalues, rsa.CRTValue(i))
	}

	return &rsa.PrivateKey{
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
	}, nil
}

func encodeCertificateAndKey(rsakey *rsa.PrivateKey, derBytes []byte) (tls.Certificate, error) {
	keyPem := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(rsakey)}
	certPem := &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}

	var keyPemBytes bytes.Buffer
	if err := pem.Encode(&keyPemBytes, keyPem); err != nil {
		return tls.Certificate{}, err
	}

	var certPemBytes bytes.Buffer
	if err := pem.Encode(&certPemBytes, certPem); err != nil {
		return tls.Certificate{}, err
	}

	return tls.X509KeyPair(certPemBytes.Bytes(), keyPemBytes.Bytes())
}

func (a *Account) handleFileVersioning(filePath string) error {
	file := File{Path: filePath}
	hash, err := file.Hash()
	if err != nil {
		return err
	}

	versionsDir := filePath + ".versions"
	if err = os.MkdirAll(versionsDir, DirPerm); err != nil {
		return err
	}

	return os.Rename(filePath, path.Join(versionsDir, hash))
}

func (a *Account) prepareEncryptionEntities(recipients []*Friend) []*openpgp.Entity {
	entities := []*openpgp.Entity{a.entity}
	for _, f := range recipients {
		entities = append(entities, f.entity)
	}
	return entities
}

func (a *Account) AddFile(r io.Reader, name string, recipients []*Friend) (*File, error) {
	if path.Ext(name) != ".pgp" {
		name += ".pgp"
	}

	fpr := a.Fingerprint().String()
	p := path.Join(a.path, fpr, name)
	
	// Create all parent directories for the file path
	if err := os.MkdirAll(path.Dir(p), DirPerm); err != nil {
		return nil, err
	}
	if err := a.handleExistingFile(p); err != nil {
		return nil, err
	}

	if err := a.writeEncryptedFile(p, r, recipients); err != nil {
		return nil, err
	}

	return &File{Path: p}, nil
}

func (a *Account) handleExistingFile(p string) error {
	if _, err := os.Stat(p); err == nil {
		return a.handleFileVersioning(p)
	}
	return nil
}

func (a *Account) writeEncryptedFile(p string, r io.Reader, recipients []*Friend) error {
	entities := a.prepareEncryptionEntities(recipients)

	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()

	w, err := openpgp.Encrypt(file, entities, a.entity, nil, nil)
	if err != nil {
		return err
	}
	if w == nil {
		return errors.New("openpgp.Encrypt returned nil writer")
	}

	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Close()
}

func (a *Account) RemoveFile(file *File) error {
	if file == nil {
		return nil
	}
	
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

// resolveFriendPath resolves a path for a friend's content, checking both followed and unfollowed directories.
// Returns the resolved path or os.ErrNotExist if neither exists.
func (a *Account) resolveFriendPath(fpr Fingerprint, subpath string) (string, error) {
	followedPath := path.Join(a.path, fpr.String(), subpath)
	unfollowedPath := path.Join(a.path, "."+fpr.String(), subpath)

	if _, err := os.Stat(followedPath); err == nil {
		return followedPath, nil
	}
	if _, err := os.Stat(unfollowedPath); err == nil {
		return unfollowedPath, nil
	}

	return "", os.ErrNotExist
}

type dirEntry struct {
	entry        fs.DirEntry
	modification time.Time
}

func filterRecentFiles(files []fs.DirEntry, after time.Time) []dirEntry {
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
	return recent
}

func sortByModificationTime(entries []dirEntry) {
	slices.SortFunc(entries, func(a, b dirEntry) int {
		if a.modification.Before(b.modification) {
			return -1
		}
		if a.modification.After(b.modification) {
			return 1
		}
		return 0
	})
}

func applyLimit(entries []dirEntry, limit uint) []dirEntry {
	if limit == 0 || uint(len(entries)) <= limit {
		return entries
	}
	return entries[:limit]
}

func (a *Account) ListFiles(fingerprint Fingerprint, after time.Time, limit uint) []*File {
	dirpath, err := a.resolveFriendPath(fingerprint, "")
	if err != nil {
		return []*File{}
	}

	files, err := os.ReadDir(dirpath)
	if err != nil {
		return []*File{}
	}

	recent := filterRecentFiles(files, after)
	sortByModificationTime(recent)
	page := applyLimit(recent, limit)

	list := make([]*File, 0, len(page))
	for _, item := range page {
		list = append(list, &File{
			Path: path.Join(dirpath, item.entry.Name()),
		})
	}

	return list
}

// syncState tracks the last successful sync time for each friend
type syncState struct {
	LastSync map[string]time.Time `json:"last_sync"`
}

func syncStateFile(d string) string { return path.Join(mauDir(d), syncStateFilename) }

// GetLastSyncTime returns the last successful sync time for a friend
// Returns zero time if no sync has occurred
func (a *Account) GetLastSyncTime(fpr Fingerprint) time.Time {
	state, err := a.loadSyncState()
	if err != nil {
		return time.Time{}
	}
	
	if lastSync, exists := state.LastSync[fpr.String()]; exists {
		return lastSync
	}
	
	return time.Time{}
}

// UpdateLastSyncTime records the time of the last successful sync for a friend
func (a *Account) UpdateLastSyncTime(fpr Fingerprint, syncTime time.Time) error {
	state, err := a.loadSyncState()
	if err != nil {
		// If file doesn't exist, create new state
		state = &syncState{
			LastSync: make(map[string]time.Time),
		}
	}
	
	state.LastSync[fpr.String()] = syncTime
	
	return a.saveSyncState(state)
}

func (a *Account) loadSyncState() (*syncState, error) {
	filePath := syncStateFile(a.path)
	
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var state syncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	
	return &state, nil
}

func (a *Account) saveSyncState(state *syncState) error {
	filePath := syncStateFile(a.path)
	
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	
	return os.WriteFile(filePath, data, FilePerm)
}
