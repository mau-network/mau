package mau

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount_AddIdentity(t *testing.T) {
	// Setup
	tmpDir := t.TempDir()
	name := "Test User"
	email := "test@example.com"
	passphrase := "test-passphrase-123"

	// Create account
	acc, err := NewAccount(tmpDir, name, email, passphrase)
	require.NoError(t, err)
	require.NotNil(t, acc)

	// Verify initial identity
	assert.Equal(t, name, acc.Name())
	assert.Equal(t, email, acc.Email())
	assert.Len(t, acc.ListIdentities(), 1)

	// Add new identity
	newName := "New Name"
	newEmail := "new@example.com"
	err = acc.AddIdentity(newName, newEmail, passphrase)
	require.NoError(t, err)

	// Verify two identities exist
	identities := acc.ListIdentities()
	assert.Len(t, identities, 2)

	// Reload account from disk
	acc2, err := OpenAccount(tmpDir, passphrase)
	require.NoError(t, err)
	require.NotNil(t, acc2)

	// Verify both identities persisted
	identities2 := acc2.ListIdentities()
	assert.Len(t, identities2, 2)
	assert.Contains(t, identities2, name+" <"+email+">")
	assert.Contains(t, identities2, newName+" <"+newEmail+">")
}

func TestAccount_AddIdentity_WrongPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	acc, err := NewAccount(tmpDir, "Test", "test@example.com", "correct-pass")
	require.NoError(t, err)

	err = acc.AddIdentity("New", "new@example.com", "wrong-pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "incorrect passphrase")
}

func TestAccount_AddIdentity_EmptyPassphrase(t *testing.T) {
	tmpDir := t.TempDir()
	acc, err := NewAccount(tmpDir, "Test", "test@example.com", "test-pass")
	require.NoError(t, err)

	err = acc.AddIdentity("New", "new@example.com", "")
	assert.ErrorIs(t, err, ErrPassphraseRequired)
}

func TestAccount_SetPrimaryIdentity(t *testing.T) {
	tmpDir := t.TempDir()
	passphrase := "test-pass"
	
	acc, err := NewAccount(tmpDir, "First", "first@example.com", passphrase)
	require.NoError(t, err)

	// Add second identity
	err = acc.AddIdentity("Second", "second@example.com", passphrase)
	require.NoError(t, err)

	// Initially first identity should be primary
	assert.Equal(t, "First", acc.Name())
	assert.Equal(t, "first@example.com", acc.Email())

	// Set second as primary
	secondIdentity := "Second <second@example.com>"
	err = acc.SetPrimaryIdentity(secondIdentity, passphrase)
	require.NoError(t, err)

	// After setting, should immediately return new primary
	assert.Equal(t, "Second", acc.Name())
	assert.Equal(t, "second@example.com", acc.Email())

	// Reload and verify it persisted
	acc2, err := OpenAccount(tmpDir, passphrase)
	require.NoError(t, err)
	
	// Primary identity should now be "Second"
	assert.Equal(t, "Second", acc2.Name())
	assert.Equal(t, "second@example.com", acc2.Email())
}

func TestAccount_ListIdentities_Empty(t *testing.T) {
	acc := &Account{}
	identities := acc.ListIdentities()
	assert.Nil(t, identities)
}

func TestAccount_FingerprintUnchanged(t *testing.T) {
	tmpDir := t.TempDir()
	passphrase := "test-pass"
	
	acc, err := NewAccount(tmpDir, "Original", "original@example.com", passphrase)
	require.NoError(t, err)

	originalFingerprint := acc.Fingerprint()

	// Add new identity
	err = acc.AddIdentity("New", "new@example.com", passphrase)
	require.NoError(t, err)

	// Fingerprint should remain the same
	assert.Equal(t, originalFingerprint, acc.Fingerprint())

	// Reload and verify fingerprint still the same
	acc2, err := OpenAccount(tmpDir, passphrase)
	require.NoError(t, err)
	assert.Equal(t, originalFingerprint, acc2.Fingerprint())
}
