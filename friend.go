package mau

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"

	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp"
	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
	"golang.org/x/crypto/openpgp/armor"
	//nolint:staticcheck // SA1019: openpgp deprecated but required for this project
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
	if f.entity == nil {
		return ""
	}
	for _, i := range f.entity.Identities {
		return i.UserId.Name
	}

	return ""
}

func (f *Friend) Email() string {
	if f.entity == nil {
		return ""
	}
	for _, i := range f.entity.Identities {
		return i.UserId.Email
	}

	return ""
}

func (f *Friend) Fingerprint() Fingerprint {
	if f.entity == nil || f.entity.PrimaryKey == nil {
		return Fingerprint{}
	}
	return f.entity.PrimaryKey.Fingerprint
}

func readFriend(account *Account, reader io.Reader) (*Friend, error) {
	keyring := openpgp.EntityList{account.entity}

	decryptedFile, err := openpgp.ReadMessage(reader, keyring, nil, nil)
	if err != nil {
		return nil, err
	}
	if decryptedFile == nil {
		return nil, fmt.Errorf("openpgp.ReadMessage returned nil")
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
	if entity == nil || entity.PrimaryKey == nil {
		return nil, fmt.Errorf("openpgp.ReadEntity returned nil or incomplete entity")
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
	if w == nil {
		return nil, fmt.Errorf("openpgp.Encrypt returned nil writer")
	}

	err = entity.Serialize(w)
	w.Close()
	if err != nil {
		return nil, err
	}

	friend := Friend{
		entity: entity,
	}

	return &friend, nil
}

func (a *Account) RemoveFriend(friend *Friend) error {
	file := fmt.Sprintf("%s.pgp", friend.Fingerprint())
	uncategorized := path.Join(mauDir(a.path), file)
	pattern := path.Join(mauDir(a.path), "**", file)

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
