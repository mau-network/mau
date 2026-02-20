package mau

import (
	"fmt"
	"os"
	"path"
)

type Keyring struct {
	Path        string
	Friends     []*Friend
	SubKeyrings []*Keyring
}

func (k *Keyring) Name() string {
	if k == nil {
		return ""
	}
	return path.Base(k.Path)
}

func (k *Keyring) FriendsSet() []*Friend {
	friends := map[Fingerprint]*Friend{}
	for _, f := range k.Friends {
		friends[f.Fingerprint()] = f
	}

	for _, sk := range k.SubKeyrings {
		for _, f := range sk.FriendsSet() {
			friends[f.Fingerprint()] = f
		}
	}

	set := []*Friend{}
	for _, f := range friends {
		set = append(set, f)
	}

	return set
}

func (k *Keyring) FindByFingerprint(fingerprint Fingerprint) *Friend {
	if k == nil {
		return nil
	}
	
	for _, friend := range k.Friends {
		if friend.Fingerprint() == fingerprint {
			return friend
		}
	}

	for _, keyring := range k.SubKeyrings {
		friend := keyring.FindByFingerprint(fingerprint)
		if friend != nil {
			return friend
		}
	}

	return nil
}

func (k *Keyring) FriendById(id uint64) *Friend {
	for _, f := range k.Friends {
		if f.entity.PrimaryKey.KeyId == id {
			return f
		}
		for _, sk := range f.entity.Subkeys {
			if sk.PublicKey.KeyId == id {
				return f
			}
		}
	}

	for _, sk := range k.SubKeyrings {
		f := sk.FriendById(id)
		if f != nil {
			return f
		}
	}

	return nil
}

func (k *Keyring) read(account *Account) error {
	files, err := os.ReadDir(k.Path)
	if err != nil {
		return fmt.Errorf("failed to read keyring directory %s: %w", k.Path, err)
	}

	for _, file := range files {
		if file.Name() == accountKeyFilename {
			continue
		}

		filePath := path.Join(k.Path, file.Name())

		if file.IsDir() {
			keyring := Keyring{Path: filePath}

			err := keyring.read(account)
			if err != nil {
				return err
			}

			k.SubKeyrings = append(k.SubKeyrings, &keyring)
			continue
		}

		reader, err := os.Open(filePath)
		if err != nil {
			return err
		}

		friend, err := readFriend(account, reader)
		reader.Close()
		if err != nil {
			return fmt.Errorf("failed to read friend key from %s: %w", filePath, err)
		}

		k.Friends = append(k.Friends, friend)
	}

	return nil
}
