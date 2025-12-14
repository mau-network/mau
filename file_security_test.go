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
	account, _ := NewAccount(accountDir, "Alice", "alice@example.com", "password")

	friendDir := t.TempDir()
	friend, _ := NewAccount(friendDir, "Bob", "bob@example.com", "password")

	// Export and add friend's key to account
	var friendKey bytes.Buffer
	friend.Export(&friendKey)
	_, _ = account.AddFriend(&friendKey)

	t.Run("Valid signature from expected signer", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		account.Export(&accountKey)
		accountFriend, _ := friend.AddFriend(&accountKey)

		// Friend creates a file encrypted for account
		file, err := friend.AddFile(strings.NewReader("Hello from Bob"), "message.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy file to account's directory
		data, _ := os.ReadFile(file.Path)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/message.txt.pgp"
		os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		os.WriteFile(accountFilePath, data, 0600)

		// Account verifies the signature from friend
		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Valid signature should verify successfully")
	})

	t.Run("File signed by friend and encrypted for account", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		account.Export(&accountKey)
		accountFriend, _ := friend.AddFriend(&accountKey)

		// Friend creates file encrypted for account
		file, err := friend.AddFile(strings.NewReader("Encrypted message"), "encrypted.txt", []*Friend{accountFriend})
		assert.NoError(t, err)

		// Copy file to account's directory for verification
		data, _ := os.ReadFile(file.Path)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/encrypted.txt.pgp"
		os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		os.WriteFile(accountFilePath, data, 0600)

		accountFile := &File{Path: accountFilePath}
		err = accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Should verify signature of encrypted and signed file")
	})

	t.Run("Files are always signed by current implementation", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		account.Export(&accountKey)
		accountFriend, _ := friend.AddFriend(&accountKey)

		// Create a file - it's always signed with current implementation
		file, _ := friend.AddFile(strings.NewReader("Message"), "test.txt", []*Friend{accountFriend})

		// Copy to account directory
		data, _ := os.ReadFile(file.Path)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/test.txt.pgp"
		os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		os.WriteFile(accountFilePath, data, 0600)

		accountFile := &File{Path: accountFilePath}
		err := accountFile.VerifySignature(account, friend.Fingerprint())
		assert.NoError(t, err, "Files are always signed by current implementation")
	})

	t.Run("Signer not in friend list", func(t *testing.T) {
		// Create another account that's not a friend
		strangerDir := t.TempDir()
		stranger, _ := NewAccount(strangerDir, "Stranger", "stranger@example.com", "password")

		file, _ := stranger.AddFile(strings.NewReader("Message from stranger"), "strange.txt", []*Friend{})

		// Try to verify with account that doesn't have stranger as friend
		err := file.VerifySignature(account, stranger.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not in friend list")
	})

	t.Run("File signed by wrong person", func(t *testing.T) {
		// Create a third person
		thirdDir := t.TempDir()
		third, _ := NewAccount(thirdDir, "Charlie", "charlie@example.com", "password")

		// Add third to account's friend list
		var thirdKey bytes.Buffer
		third.Export(&thirdKey)
		_, _ = account.AddFriend(&thirdKey)

		// Get account's public key
		var accountKey bytes.Buffer
		account.Export(&accountKey)
		accountFriend, _ := friend.AddFriend(&accountKey)

		// Friend signs a file encrypted for account
		file, _ := friend.AddFile(strings.NewReader("From Bob"), "bob.txt", []*Friend{accountFriend})

		// Copy to account directory
		data, _ := os.ReadFile(file.Path)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/bob.txt.pgp"
		os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		os.WriteFile(accountFilePath, data, 0600)

		accountFile := &File{Path: accountFilePath}
		// Try to verify expecting a different signer (third instead of friend)
		// Both friend and third are in account's friend list, but file was signed by friend
		err := accountFile.VerifySignature(account, third.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signed by unexpected key")
	})

	t.Run("Corrupted file data", func(t *testing.T) {
		// Get account's public key
		var accountKey bytes.Buffer
		account.Export(&accountKey)
		accountFriend, _ := friend.AddFriend(&accountKey)

		file, _ := friend.AddFile(strings.NewReader("Original message"), "original.txt", []*Friend{accountFriend})

		// Copy to account directory
		data, _ := os.ReadFile(file.Path)
		accountFilePath := accountDir + "/" + account.Fingerprint().String() + "/original.txt.pgp"
		os.MkdirAll(accountDir+"/"+account.Fingerprint().String(), 0700)
		os.WriteFile(accountFilePath, data, 0600)

		// Corrupt the file
		data[len(data)/2] ^= 0xFF // Flip bits in the middle
		os.WriteFile(accountFilePath, data, 0600)

		accountFile := &File{Path: accountFilePath}
		err := accountFile.VerifySignature(account, friend.Fingerprint())
		assert.Error(t, err, "Corrupted file should fail verification")
	})

	t.Run("Nonexistent file", func(t *testing.T) {
		file := &File{Path: "/nonexistent/file.pgp"}
		err := file.VerifySignature(account, friend.Fingerprint())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read file")
	})
}
