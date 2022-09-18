package main

import (
	"fmt"
	"os"
	"path"
)

func ListFollows(account *Account) ([]*Friend, error) {
	files, err := os.ReadDir(account.path)
	if err != nil {
		return nil, err
	}

	keyring, err := ListFriends(account)
	if err != nil {
		return nil, fmt.Errorf("Error listing friends: %w", err)
	}

	follows := []*Friend{}
	for _, file := range files {
		if file.IsDir() && file.Name()[0] != '.' {
			fpr, err := ParseFingerprint(file.Name())
			if err != nil {
				return follows, fmt.Errorf("Error parsing fingerprint: %w", err)
			}

			friend := keyring.FindByFingerprint(fpr)
			if friend != nil {
				follows = append(follows, friend)
			}
		}
	}

	return follows, nil
}

func Follow(account *Account, friend *Friend) error {
	fpr := friend.Fingerprint().String()
	unfollowed := path.Join(account.path, "."+fpr)
	followed := path.Join(account.path, fpr)

	if _, err := os.Stat(followed); err == nil {
		return nil
	}

	if _, err := os.Stat(unfollowed); err == nil {
		return os.Rename(unfollowed, followed)
	}

	return os.Mkdir(followed, 0700)
}

func Unfollow(account *Account, friend *Friend) error {
	fpr := friend.Fingerprint().String()
	unfollowed := path.Join(account.path, "."+fpr)
	followed := path.Join(account.path, fpr)

	if _, err := os.Stat(followed); err == nil {
		return os.Rename(followed, unfollowed)
	}

	return nil
}
