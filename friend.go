package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/keybase/go-crypto/openpgp"
	"github.com/keybase/go-crypto/openpgp/armor"
	"github.com/keybase/go-crypto/openpgp/packet"
)

type Friend struct {
	entity *openpgp.Entity
}

func (f *Friend) Identity() (string, error) {
	for name := range f.entity.Identities {
		return name, nil
	}

	return "", ErrNoIdentity
}

func (f *Friend) Name() string {
	for _, i := range f.entity.Identities {
		return i.UserId.Name
	}

	return ""
}

func (f *Friend) Email() string {
	for _, i := range f.entity.Identities {
		return i.UserId.Email
	}

	return ""
}

func (f *Friend) Fingerprint() string {
	return fmt.Sprintf("%X", f.entity.PrimaryKey.Fingerprint)
}

func readFriend(reader io.Reader) (*Friend, error) {
	entity, err := openpgp.ReadEntity(packet.NewReader(reader))
	if err != nil {
		return nil, err
	}

	return &Friend{entity: entity}, nil
}

// TODO Encrypt the friends public key to hide identities
func AddFriend(account *Account, reader io.Reader) (*Friend, error) {
	block, err := armor.Decode(reader)
	if err != nil {
		return nil, err
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		return nil, err
	}

	f, err := os.Create(fmt.Sprintf("%s/%X.pgp", mauDir(account.path), entity.PrimaryKey.Fingerprint))
	if err != nil {
		return nil, err
	}

	err = entity.Serialize(f)
	if err != nil {
		return nil, err
	}

	friend := Friend{
		entity: entity,
	}

	return &friend, nil
}

func RemoveFriend(account *Account, friend *Friend) error {
	file := fmt.Sprintf("%s.pgp", friend.Fingerprint())
	uncategorized := fmt.Sprintf("%s/%s", mauDir(account.path), file)
	pattern := fmt.Sprintf("%s/**/%s", mauDir(account.path), file)

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if _, err := os.Stat(uncategorized); err == nil {
		matches = append(matches, uncategorized)
	}

	for _, match := range matches {
		err = os.Remove(match)
		if err != nil {
			return err
		}
	}

	return Unfollow(account, friend)
}

func ListFriends(account *Account) (*KeyRing, error) {
	friends := KeyRing{Path: mauDir(account.path)}

	err := friends.read()
	if err != nil {
		return nil, err
	}

	return &friends, nil
}
