package main

import (
	"fmt"
	"os"
	"path"
)

type KeyRing struct {
	Path     string
	Friends  []*Friend
	KeyRings []*KeyRing
}

func (k *KeyRing) Name() string {
	return path.Base(k.Path)
}

func (k *KeyRing) FriendsSet() []*Friend {
	friends := map[Fingerprint]*Friend{}
	for _, f := range k.Friends {
		friends[f.Fingerprint()] = f
	}

	for _, sk := range k.KeyRings {
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

func (k *KeyRing) FindByFingerprint(fingerprint Fingerprint) *Friend {
	for _, friend := range k.Friends {
		if friend.Fingerprint() == fingerprint {
			return friend
		}
	}

	for _, keyring := range k.KeyRings {
		friend := keyring.FindByFingerprint(fingerprint)
		if friend != nil {
			return friend
		}
	}

	return nil
}

func (k *KeyRing) FriendById(id uint64) *Friend {
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

	for _, sk := range k.KeyRings {
		f := sk.FriendById(id)
		if f != nil {
			return f
		}
	}

	return nil
}

func (k *KeyRing) read() error {
	files, err := os.ReadDir(k.Path)
	if err != nil {
		return err
	}

	for _, file := range files {
		if file.Name() == accountKeyFilename {
			continue
		}

		if file.IsDir() {
			keyring := KeyRing{Path: fmt.Sprintf("%s/%s", k.Path, file.Name())}

			err := keyring.read()
			if err != nil {
				return err
			}

			k.KeyRings = append(k.KeyRings, &keyring)
			continue
		}

		reader, err := os.Open(fmt.Sprintf("%s/%s", k.Path, file.Name()))
		if err != nil {
			return err
		}

		friend, err := readFriend(reader)
		reader.Close()
		if err != nil {
			return err
		}

		k.Friends = append(k.Friends, friend)
	}

	return nil
}
