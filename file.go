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
func validateFileName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty filename", ErrInvalidFileName)
	}

	// Check for path separators
	if strings.Contains(name, string(filepath.Separator)) || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("%w: contains path separators", ErrInvalidFileName)
	}

	// Check for relative path components
	if name == "." || name == ".." || strings.HasPrefix(name, "."+string(filepath.Separator)) || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%w: contains relative path components", ErrInvalidFileName)
	}

	return nil
}

type File struct {
	Path    string
	version bool
}

func (f *File) Name() string {
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

	versions := []*File{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		versions = append(versions, &File{
			Path:    path.Join(vPath, entry.Name()),
			version: true,
		})
	}

	return versions
}

// VerifySignature verifies that a file was signed by the expected peer and is encrypted for the account
func (f *File) VerifySignature(account *Account, expectedSigner Fingerprint) error {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		return fmt.Errorf("failed to read file for verification: %w", err)
	}

	// Get all friends to build complete keyring for signature verification
	friends, err := account.ListFriends()
	if err != nil {
		return fmt.Errorf("failed to list friends for signature verification: %w", err)
	}

	// Verify expected signer is in friend list
	expectedSignerFriend := friends.FindByFingerprint(expectedSigner)
	if expectedSignerFriend == nil {
		return fmt.Errorf("signer %s not in friend list", expectedSigner)
	}

	// Create keyring with account's private key (for decryption) and all friends' public keys (for signature verification)
	keyring := openpgp.EntityList{account.entity}
	for _, friend := range friends.FriendsSet() {
		keyring = append(keyring, friend.entity)
	}

	// Try to read and verify the message
	md, err := openpgp.ReadMessage(bytes.NewReader(data), keyring, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to read OpenPGP message: %w", err)
	}

	// Check if message is signed
	if !md.IsSigned {
		return errors.New("file is not signed")
	}

	// Read the entire message to trigger signature verification
	_, err = io.ReadAll(md.UnverifiedBody)
	if err != nil {
		return fmt.Errorf("failed to read message body: %w", err)
	}

	// Check signature validity
	if md.SignatureError != nil {
		return fmt.Errorf("invalid signature: %w", md.SignatureError)
	}

	// Verify the signer is who we expect
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

func (f *File) Recipients(account *Account) ([]*Friend, error) {
	r, err := os.Open(f.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	k, err := account.ListFriends()
	if err != nil {
		return nil, err
	}

	packets := packet.NewReader(r)
	keysIDs := []uint64{}

	for {
		p, err := packets.Next()
		if err != nil {
			break
		}

		switch p := p.(type) {
		case *packet.EncryptedKey:
			keysIDs = append(keysIDs, p.KeyId)
		}
	}

	recipients := []*Friend{}
	for _, id := range keysIDs {
		friend := k.FriendById(id)
		if friend != nil {
			recipients = append(recipients, friend)
		}
	}

	return recipients, nil
}

func (f *File) Reader(account *Account) (io.Reader, error) {
	r, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}

	keyring := openpgp.EntityList{account.entity}
	decryptedFile, err := openpgp.ReadMessage(bytes.NewReader(r), keyring, nil, nil)
	if err != nil {
		return nil, err
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
