package main

import (
	"os"
	"path"
)

func ListFollows(account *Account) ([]*Friend, error) {
	files, err := os.ReadDir(accountRoot)
	if err != nil {
		return nil, err
	}

	keyring, err := ListFriends(account)
	if err != nil {
		return nil, err
	}

	follows := []*Friend{}
	for _, file := range files {
		if file.IsDir() && file.Name()[0] != '.' {
			friend := keyring.FindByFingerprint(file.Name())
			if friend != nil {
				follows = append(follows, friend)
			}
		}
	}

	return follows, nil
}

func Follow(account *Account, friend *Friend) error {
	unfollowed := path.Join(accountRoot, "."+friend.Fingerprint())
	followed := path.Join(accountRoot, friend.Fingerprint())

	if _, err := os.Stat(followed); err == nil {
		return nil
	}

	if _, err := os.Stat(unfollowed); err == nil {
		return os.Rename(unfollowed, followed)
	}

	return os.Mkdir(followed, 0700)
}

func Unfollow(account *Account, friend *Friend) error {
	unfollowed := path.Join(accountRoot, "."+friend.Fingerprint())
	followed := path.Join(accountRoot, friend.Fingerprint())

	if _, err := os.Stat(followed); err == nil {
		return os.Rename(followed, unfollowed)
	}

	return nil
}
