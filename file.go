package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path"
	"sort"
	"time"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/packet"
	"github.com/multiformats/go-multihash"
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

	k, err := ListFriends(account)
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

	hash, err := multihash.Sum(content, multihash.SHA2_256, -1)
	if err != nil {
		return "", err
	}

	return hash.B58String(), nil
}

func (f *File) Size() (int64, error) {
	info, err := os.Stat(f.Path)
	if err != nil {
		return 0, err
	}

	return info.Size(), nil
}

func (f *File) Deleted(account *Account) bool {
	r, err := f.Reader(account)
	if err != nil {
		return false
	}

	buf := make([]byte, 1)
	_, err = io.ReadFull(r, buf)

	return err != nil
}

func GetVersion(account *Account, fpr, name, version string) (*File, error) {
	followedPath := path.Join(accountRoot, fpr, name+".versions", version)
	unfollowedPath := path.Join(accountRoot, "."+fpr, name+".versions", version)
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

func GetFile(account *Account, fpr, name string) (*File, error) {
	followedPath := path.Join(accountRoot, fpr, name)
	unfollowedPath := path.Join(accountRoot, "."+fpr, name)
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

func AddFile(account *Account, r io.Reader, name string, recipients []*Friend) (*File, error) {
	if path.Ext(name) != ".pgp" {
		name += ".pgp"
	}

	fpr := account.Fingerprint()

	if err := ensureDirectory(path.Join(accountRoot, fpr)); err != nil {
		return nil, err
	}

	p := path.Join(accountRoot, fpr, name)
	if _, err := os.Stat(p); err == nil {
		file := File{Path: p}
		np, err := file.Hash()
		if err != nil {
			return nil, err
		}

		if err = ensureDirectory(p + ".versions"); err != nil {
			return nil, err
		}

		if err = os.Rename(p, path.Join(p+".versions", np)); err != nil {
			return nil, err
		}
	}

	entities := []*openpgp.Entity{account.entity}
	for _, f := range recipients {
		entities = append(entities, f.entity)
	}

	file, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	w, err := openpgp.Encrypt(file, entities, account.entity, nil, nil)
	if err != nil {
		return nil, err
	}

	io.Copy(w, r)
	w.Close()

	return &File{Path: p}, nil
}

func RemoveFile(account *Account, file *File) error {
	rs, err := file.Recipients(account)
	if err != nil {
		return err
	}

	_, err = AddFile(account, bytes.NewReader([]byte{}), file.Name(), rs)
	return err
}

func ListFiles(account *Account, fingerprint string, after time.Time, limit uint) []*File {
	followedPath := path.Join(accountRoot, fingerprint)
	unfollowedPath := path.Join(accountRoot, "."+fingerprint)
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
