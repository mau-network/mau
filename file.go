package mau

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

type File struct {
	Path    string
	version bool
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

func (f *File) Name() string {
	return path.Base(f.Path)
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
	followedPath := path.Join(a.path, fpr.String(), name+".versions", version)
	unfollowedPath := path.Join(a.path, "."+fpr.String(), name+".versions", version)
	var filepath string

	if _, err := os.Stat(followedPath); err == nil {
		filepath = followedPath
	} else if _, err := os.Stat(unfollowedPath); err == nil {
		filepath = unfollowedPath
	} else {
		return nil, os.ErrNotExist
	}

	return &File{Path: filepath, version: true}, nil
}

func (a *Account) GetFile(fpr Fingerprint, name string) (*File, error) {
	followedPath := path.Join(a.path, fpr.String(), name)
	unfollowedPath := path.Join(a.path, "."+fpr.String(), name)
	var filepath string

	if _, err := os.Stat(followedPath); err == nil {
		filepath = followedPath
	} else if _, err := os.Stat(unfollowedPath); err == nil {
		filepath = unfollowedPath
	} else {
		return nil, os.ErrNotExist
	}

	return &File{Path: filepath}, nil
}
