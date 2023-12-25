package mau

import (
	"fmt"
	"os"
	"path"
)

func (a *Account) ListFollows() ([]*Friend, error) {
	files, err := os.ReadDir(a.path)
	if err != nil {
		return nil, err
	}

	keyring, err := a.ListFriends()
	if err != nil {
		return nil, fmt.Errorf("Error listing friends: %w", err)
	}

	follows := []*Friend{}
	for _, file := range files {
		if file.IsDir() && file.Name()[0] != '.' {
			fpr, err := FingerprintFromString(file.Name())
			if err != nil {
				continue // Ignore any directory that's not a fingerprint
			}

			friend := keyring.FindByFingerprint(fpr)
			if friend != nil {
				follows = append(follows, friend)
			}
		}
	}

	return follows, nil
}

func (a *Account) Follow(friend *Friend) error {
	fpr := friend.Fingerprint().String()
	unfollowed := path.Join(a.path, "."+fpr)
	followed := path.Join(a.path, fpr)

	if _, err := os.Stat(followed); err == nil {
		return nil
	}

	if _, err := os.Stat(unfollowed); err == nil {
		return os.Rename(unfollowed, followed)
	}

	return os.Mkdir(followed, dirPerm)
}

func (a *Account) Unfollow(friend *Friend) error {
	fpr := friend.Fingerprint().String()
	unfollowed := path.Join(a.path, "."+fpr)
	followed := path.Join(a.path, fpr)

	if _, err := os.Stat(followed); err == nil {
		return os.Rename(followed, unfollowed)
	}

	return nil
}
