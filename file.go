package mau

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp"
	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp/packet"
)

var (
	ErrInvalidFileName = errors.New("invalid file name: contains path separators or invalid characters")
)

// validateFileName checks if a filename is safe and doesn't contain path traversal attempts
func containsPathSeparator(name string) bool {
	return strings.Contains(name, string(filepath.Separator)) ||
		strings.Contains(name, "/") ||
		strings.Contains(name, "\\")
}

func isRelativePathComponent(name string) bool {
	return name == "." ||
		name == ".." ||
		strings.HasPrefix(name, "."+string(filepath.Separator)) ||
		strings.HasPrefix(name, ".."+string(filepath.Separator))
}

func validateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty filename", ErrInvalidFileName)
	}

	// Check for path separators
	if containsPathSeparator(name) {
		return fmt.Errorf("%w: contains path separators", ErrInvalidFileName)
	}

	// Check for relative path components
	if isRelativePathComponent(name) {
		return fmt.Errorf("%w: contains relative path components", ErrInvalidFileName)
	}

	return nil
}

type File struct {
	Path    string
	version bool
}

func (f *File) Name() string {
	if f == nil {
		return ""
	}
	return path.Base(f.Path)
}

func (f *File) Versions() []*File {
	if f.version {
		return []*File{}
	}

	vPath := f.Path + ".versions"
	if _, err := os.Stat(vPath); err != nil {
		return []*File{}
	}

	entries, err := os.ReadDir(vPath)
	if err != nil {
		return []*File{}
	}

	return f.collectVersionFiles(vPath, entries)
}

func (f *File) collectVersionFiles(vPath string, entries []os.DirEntry) []*File {
	versions := []*File{}
	for _, entry := range entries {
		if !entry.IsDir() {
			versions = append(versions, &File{
				Path:    path.Join(vPath, entry.Name()),
				version: true,
			})
		}
	}
	return versions
}

func buildVerificationKeyring(account *Account, friends *Keyring) openpgp.EntityList {
	keyring := openpgp.EntityList{account.entity}
	for _, friend := range friends.FriendsSet() {
		keyring = append(keyring, friend.entity)
	}
	return keyring
}

func verifyExpectedSigner(friends *Keyring, expectedSigner Fingerprint) error {
	expectedSignerFriend := friends.FindByFingerprint(expectedSigner)
	if expectedSignerFriend == nil {
		return fmt.Errorf("signer %s not in friend list", expectedSigner)
	}
	return nil
}

func readAndVerifyMessage(data []byte, keyring openpgp.EntityList) (*openpgp.MessageDetails, error) {
	md, err := openpgp.ReadMessage(bytes.NewReader(data), keyring, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to read OpenPGP message: %w", err)
	}
	if md == nil {
		return nil, errors.New("ReadMessage returned nil message details")
	}

	if !md.IsSigned {
		return nil, errors.New("file is not signed")
	}

	if err := verifyMessageSignature(md); err != nil {
		return nil, err
	}

	return md, nil
}

func verifyMessageSignature(md *openpgp.MessageDetails) error {
	_, err := io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return fmt.Errorf("failed to read message body: %w", err)
	}

	if md.SignatureError != nil {
		return fmt.Errorf("invalid signature: %w", md.SignatureError)
	}

	return nil
}

func checkSignerIdentity(md *openpgp.MessageDetails, expectedSigner Fingerprint) error {
	if md.SignedBy == nil || md.SignedBy.PublicKey == nil {
		return errors.New("no valid signature found")
	}

	actualSigner := Fingerprint(md.SignedBy.PublicKey.Fingerprint)
	if actualSigner != expectedSigner {
		return fmt.Errorf("file signed by unexpected key: got %s, expected %s",
			actualSigner, expectedSigner)
	}

	return nil
}

// VerifySignature verifies that a file was signed by the expected peer and is encrypted for the account
func (f *File) VerifySignature(account *Account, expectedSigner Fingerprint) error {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("failed to read file for verification: %w", err)
	}

	friends, err := account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends for signature verification: %w", err)
	}

	if err := verifyExpectedSigner(friends, expectedSigner); err != nil {
		return err
	}

	return f.verifyMessageSignatureAndSigner(account, friends, data, expectedSigner)
}

func (f *File) verifyMessageSignatureAndSigner(account *Account, friends *Keyring, data []byte, expectedSigner Fingerprint) error {
	keyring := buildVerificationKeyring(account, friends)
	md, err := readAndVerifyMessage(data, keyring)
	if err != nil {
		return err
	}
	if md == nil {
		return errors.New("openpgp.ReadMessage returned nil")
	}

	return checkSignerIdentity(md, expectedSigner)
}

func extractEncryptedKeyIDs(r io.Reader) ([]uint64, error) {
	packets := packet.NewReader(r)
	keysIDs := []uint64{}

	for {
		p, err := packets.Next()
		if err != nil {
			break
		}

		if encKey, ok := p.(*packet.EncryptedKey); ok {
			keysIDs = append(keysIDs, encKey.KeyId)
		}
	}

	return keysIDs, nil
}

func friendsByKeyIDs(keyring *Keyring, keyIDs []uint64) []*Friend {
	recipients := []*Friend{}
	for _, id := range keyIDs {
		if friend := keyring.FriendById(id); friend != nil {
			recipients = append(recipients, friend)
		}
	}
	return recipients
}

func (f *File) Recipients(account *Account) ([]*Friend, error) {
	if account == nil {
		return nil, errors.New("account cannot be nil")
	}

	r, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	k, err := account.ListFriends()
	if err != nil {
		return nil, err
	}

	keysIDs, err := extractEncryptedKeyIDs(r)
	if err != nil {
		return nil, err
	}

	return friendsByKeyIDs(k, keysIDs), nil
}

func (f *File) Reader(account *Account) (io.Reader, error) {
	if f == nil {
		return nil, errors.New("file cannot be nil")
	}
	if account == nil {
		return nil, errors.New("account cannot be nil")
	}

	r, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}

	return f.decryptFileContent(r, account)
}

func (f *File) decryptFileContent(data []byte, account *Account) (io.Reader, error) {
	keyring := openpgp.EntityList{account.entity}
	decryptedFile, err := openpgp.ReadMessage(bytes.NewReader(data), keyring, nil, nil)
	if err != nil {
		return nil, err
	}
	if decryptedFile == nil {
		return nil, errors.New("openpgp.ReadMessage returned nil")
	}

	return decryptedFile.UnverifiedBody, nil
}

func (f *File) Hash() (string, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return "", err
	}

	content, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(content)

	return fmt.Sprintf("%x", hash), nil
}

func (f *File) Size() (int64, error) {
	info, err := os.Stat(f.Path)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (f *File) Deleted() bool {
	_, err := os.Stat(f.Path)
	return os.IsNotExist(err)
}

func (a *Account) GetFileVersion(fpr Fingerprint, name, version string) (*File, error) {
	if err := validateFileName(name); err != nil {
		return nil, fmt.Errorf("invalid file name: %w", err)
	}
	if err := validateFileName(version); err != nil {
		return nil, fmt.Errorf("invalid version name: %w", err)
	}

	filepath, err := a.resolveFriendPath(fpr, path.Join(name+".versions", version))
	if err != nil {
		return nil, err
	}

	return &File{Path: filepath, version: true}, nil
}

func (a *Account) GetFile(fpr Fingerprint, name string) (*File, error) {
	if err := validateFileName(name); err != nil {
		return nil, fmt.Errorf("invalid file name: %w", err)
	}

	filepath, err := a.resolveFriendPath(fpr, name)
	if err != nil {
		return nil, err
	}

	return &File{Path: filepath}, nil
}
