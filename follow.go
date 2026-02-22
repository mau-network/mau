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
		return nil, fmt.Errorf("failed to list friends while getting follows: %w", err)
	}

	return a.collectFollowedFriends(files, keyring), nil
}

func (a *Account) collectFollowedFriends(files []os.DirEntry, keyring *Keyring) []*Friend {
	follows := []*Friend{}
	for _, file := range files {
		if file.IsDir() && file.Name()[0] != '.' {
			if friend := a.findFriendByDirName(file.Name(), keyring); friend != nil {
				follows = append(follows, friend)
			}
		}
	}
	return follows
}

func (a *Account) findFriendByDirName(name string, keyring *Keyring) *Friend {
	fpr, err := FingerprintFromString(name)
	if err != nil {
		return nil // Ignore any directory that's not a fingerprint
	}
	return keyring.FindByFingerprint(fpr)
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

	return os.Mkdir(followed, DirPerm)
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
