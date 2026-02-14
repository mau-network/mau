package mau

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVerifySignature(t *testing.T) {
	// Setup: Create two accounts
	accountDir := t.TempDir()
	account, err := NewAccount(accountDir, "Alice", "alice@example.com", "password")
	assert.NoError(t, err)

	friendDir := t.TempDir()
	friend, err := NewAccount(friendDir, "Bob", "bob@example.com", "password")
	assert.NoError(t, err)

	// Export and add friend's key to account
	var friendKey bytes.Buffer
	err = friend.Export(&friendKey)
	assert.NoError(t, err)
	_, err = account.AddFriend(&friendKey)
	assert.NoError(t, err)

	t.Run("Valid signature from expected signer", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		err := account.Export(&accountKey)
		assert.NoError(t, err)
		accountFriend, err := friend.AddFriend(&accountKey)
		assert.NoError(t, err)

		// Friend creates a file encrypted for account
		file, err := friend.AddFile(strings.NewReader("Hello from Bob"), "message.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy file to account's directory
		data, err := os.ReadFile(file.Path)
		assert.NoError(t, err)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/message.txt.pgp"
		err = os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		assert.NoError(t, err)
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		// Account verifies the signature from friend
		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Valid signature should verify successfully")
	})

	t.Run("File signed by friend and encrypted for account", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		err := account.Export(&accountKey)
		assert.NoError(t, err)
		accountFriend, err := friend.AddFriend(&accountKey)
		assert.NoError(t, err)

		// Friend creates file encrypted for account
		file, err := friend.AddFile(strings.NewReader("Encrypted message"), "encrypted.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy file to account's directory for verification
		data, err := os.ReadFile(file.Path)
		assert.NoError(t, err)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/encrypted.txt.pgp"
		err = os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		assert.NoError(t, err)
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Should verify signature of encrypted and signed file")
	})

	t.Run("Files are always signed by current implementation", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		err := account.Export(&accountKey)
		assert.NoError(t, err)
		accountFriend, err := friend.AddFriend(&accountKey)
		assert.NoError(t, err)

		// Create a file - it's always signed with current implementation
		file, err := friend.AddFile(strings.NewReader("Message"), "test.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy to account directory
		data, err := os.ReadFile(file.Path)
		assert.NoError(t, err)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/test.txt.pgp"
		err = os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		assert.NoError(t, err)
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Files are always signed by current implementation")
	})

	t.Run("Signer not in friend list", func(t *testing.T) {
		// Create another account that's not a friend
		strangerDir := t.TempDir()
		stranger, err := NewAccount(strangerDir, "Stranger", "stranger@example.com", "password")
		assert.NoError(t, err)

		file, err := stranger.AddFile(strings.NewReader("Message from stranger"), "strange.txt", []*Friend{})
		assert.NoError(t, err)

		// Try to verify with account that doesn't have stranger as friend
		err = file.VerifySignature(account, stranger.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in friend list")
	})

	t.Run("File signed by wrong person", func(t *testing.T) {
		// Create a third person
		thirdDir := t.TempDir()
		third, err := NewAccount(thirdDir, "Charlie", "charlie@example.com", "password")
		assert.NoError(t, err)

		// Add third to account's friend list
		var thirdKey bytes.Buffer
		err = third.Export(&thirdKey)
		assert.NoError(t, err)
		_, err = account.AddFriend(&thirdKey)
		assert.NoError(t, err)

		// Get account's public key
		var accountKey bytes.Buffer
		err = account.Export(&accountKey)
		assert.NoError(t, err)
		accountFriend, err := friend.AddFriend(&accountKey)
		assert.NoError(t, err)

		// Friend signs a file encrypted for account
		file, err := friend.AddFile(strings.NewReader("From Bob"), "bob.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy to account directory
		data, err := os.ReadFile(file.Path)
		assert.NoError(t, err)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/bob.txt.pgp"
		err = os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		assert.NoError(t, err)
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		accountFile := &File{Path: accountFilePath}
		// Try to verify expecting a different signer (third instead of friend)
		// Both friend and third are in account's friend list, but file was signed by friend
		err = accountFile.VerifySignature(account, third.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signed by unexpected key")
	})

	t.Run("Corrupted file data", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		err := account.Export(&accountKey)
		assert.NoError(t, err)
		accountFriend, err := friend.AddFriend(&accountKey)
		assert.NoError(t, err)

		file, err := friend.AddFile(strings.NewReader("Original message"), "original.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy to account directory
		data, err := os.ReadFile(file.Path)
		assert.NoError(t, err)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/original.txt.pgp"
		err = os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		assert.NoError(t, err)
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		// Corrupt the file
		data[len(data)/2] ^= 0xFF // Flip bits in the middle
		err = os.WriteFile(accountFilePath, data, 0600)
		assert.NoError(t, err)

		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.Error(t, err, "Corrupted file should fail verification")
	})

	t.Run("Nonexistent file", func(t *testing.T) {
		file := &File{Path: "/nonexistent/file.pgp"}
		err := file.VerifySignature(account, friend.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})
}
