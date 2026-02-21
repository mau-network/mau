package mau

import (
	"bytes"
	"crypto"
	"errors"
	"io"
	"os"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

// AddIdentity adds a new identity (name/email) to an existing account.
// The fingerprint remains unchanged since it's derived from the public key.
// The new identity is self-signed with the account's private key.
// Requires the account passphrase to save the updated entity.
func (a *Account) AddIdentity(name, email, passphrase string) error {
	if a == nil || a.entity == nil {
		return errors.New("account or entity is nil")
	}

	if a.entity.PrivateKey == nil {
		return errors.New("private key is required to sign new identity")
	}

	if a.entity.PrivateKey.Encrypted {
		return errors.New("private key must be decrypted")
	}

	// Create new UserId packet
	userId := packet.NewUserId(name, "", email)
	if userId == nil {
		return errors.New("failed to create UserId")
	}

	// Create self-signature for the new identity
	sig := &packet.Signature{
		SigType:      packet.SigTypePositiveCert,
		PubKeyAlgo:   a.entity.PrimaryKey.PubKeyAlgo,
		Hash:         crypto.SHA256,
		CreationTime: time.Now(),
		IssuerKeyId:  &a.entity.PrimaryKey.KeyId,
	}

	// Sign the UserId with the private key
	err := sig.SignUserId(userId.Id, a.entity.PrimaryKey, a.entity.PrivateKey, nil)
	if err != nil {
		return err
	}

	// Create the new Identity
	identity := &openpgp.Identity{
		Name:          userId.Id,
		UserId:        userId,
		SelfSignature: sig,
	}

	// Add to the entity's identities map
	a.entity.Identities[userId.Id] = identity

	// Mark as primary (first identity)
	if len(a.entity.Identities) == 1 {
		isPrimary := true
		identity.SelfSignature.IsPrimaryId = &isPrimary
	}

	// Save the updated entity back to disk
	return a.saveEntity(passphrase)
}

// SetPrimaryIdentity marks a specific identity as primary.
// The identity must already exist in the account.
// Requires the account passphrase to save the updated entity.
func (a *Account) SetPrimaryIdentity(identityName, passphrase string) error {
	if a == nil || a.entity == nil {
		return errors.New("account or entity is nil")
	}

	identity, exists := a.entity.Identities[identityName]
	if !exists {
		return errors.New("identity not found: " + identityName)
	}

	// Unmark all identities first
	for _, ident := range a.entity.Identities {
		if ident.SelfSignature.IsPrimaryId != nil {
			notPrimary := false
			ident.SelfSignature.IsPrimaryId = &notPrimary
		}
	}

	// Mark the selected identity as primary
	isPrimary := true
	identity.SelfSignature.IsPrimaryId = &isPrimary

	return a.saveEntity(passphrase)
}

// ListIdentities returns all identities associated with the account
func (a *Account) ListIdentities() []string {
	if a == nil || a.entity == nil {
		return nil
	}

	identities := make([]string, 0, len(a.entity.Identities))
	for name := range a.entity.Identities {
		identities = append(identities, name)
	}
	return identities
}

// saveEntity saves the updated entity back to the encrypted account file
func (a *Account) saveEntity(passphrase string) error {
	if len(passphrase) == 0 {
		return ErrPassphraseRequired
	}

	// Verify passphrase by attempting to open the account
	// This is the correct way to validate - try to decrypt the key
	testAccount, err := OpenAccount(a.path, passphrase)
	if err != nil {
		return errors.New("incorrect passphrase")
	}
	// Ensure we're not leaking the test account
	_ = testAccount

	// Serialize entity to buffer
	var buf bytes.Buffer
	err = a.entity.SerializePrivate(&buf, nil)
	if err != nil {
		return err
	}

	// Write encrypted to account file (atomic write pattern)
	acc := accountFile(a.path)
	tmpFile := acc + ".tmp"
	plainFile, err := os.Create(tmpFile)
	if err != nil {
		return err
	}

	encryptedFile, err := openpgp.SymmetricallyEncrypt(plainFile, []byte(passphrase), nil, nil)
	if err != nil {
		plainFile.Close()
		os.Remove(tmpFile)
		return err
	}

	_, err = io.Copy(encryptedFile, &buf)
	encryptedFile.Close()
	plainFile.Close()
	
	if err != nil {
		os.Remove(tmpFile)
		return err
	}

	// Atomic rename
	return os.Rename(tmpFile, acc)
}

