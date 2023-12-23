package mau

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
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

func (f *Friend) Fingerprint() Fingerprint {
	return f.entity.PrimaryKey.Fingerprint
}

func readFriend(account *Account, reader io.Reader) (*Friend, error) {
	keyring := openpgp.EntityList{account.entity}

	decryptedFile, err := openpgp.ReadMessage(reader, keyring, nil, nil)
	if err != nil {
		return nil, err
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(decryptedFile.UnverifiedBody))
	if err != nil {
		return nil, err
	}

	return &Friend{entity: entity}, nil
}

func (a *Account) AddFriend(reader io.Reader) (*Friend, error) {
	block, err := armor.Decode(reader)
	if err != nil {
		return nil, err
	}

	entity, err := openpgp.ReadEntity(packet.NewReader(block.Body))
	if err != nil {
		return nil, err
	}

	fpr := Fingerprint(entity.PrimaryKey.Fingerprint).String()
	entities := []*openpgp.Entity{a.entity}

	filePath := path.Join(mauDir(a.path), fpr+".pgp")
	file, err := os.Create(filePath)
	if err != nil {
		return nil, err
	}

	w, err := openpgp.Encrypt(file, entities, a.entity, nil, nil)
	if err != nil {
		return nil, err
	}

	entity.Serialize(w)
	w.Close()

	friend := Friend{
		entity: entity,
	}

	return &friend, nil
}

func (a *Account) RemoveFriend(friend *Friend) error {
	file := fmt.Sprintf("%s.pgp", friend.Fingerprint())
	uncategorized := fmt.Sprintf("%s/%s", mauDir(a.path), file)
	pattern := fmt.Sprintf("%s/**/%s", mauDir(a.path), file)

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

	return a.Unfollow(friend)
}

func (a *Account) ListFriends() (*Keyring, error) {
	friends := Keyring{Path: mauDir(a.path)}

	err := friends.read(a)
	if err != nil {
		return nil, err
	}

	return &friends, nil
}
