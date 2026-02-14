package mau

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddFriend(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	fingerprint := friend_account.Fingerprint()
	var friend_account_pub bytes.Buffer
	if err := friend_account.Export(&friend_account_pub); err != nil {
		t.Fatalf("Failed to export friend account: %v", err)
	}
	friend, err := account.AddFriend(&friend_account_pub)
	assert.NoError(t, err)
	assert.FileExists(t, path.Join(dir, ".mau", fingerprint.String()+".pgp"))

	t.Run("Email", func(t T) {
		assert.Equal(t, "mohamed@example.com", friend.Email())
	})

	t.Run("Name", func(t T) {
		assert.Equal(t, "Mohamed Mahmoud", friend.Name())
	})

	t.Run("Fingerprint", func(t T) {
		assert.Equal(t, friend_account.Fingerprint(), friend.Fingerprint())
	})

	t.Run("Identity", func(t T) {
		friend_account_identity, _ := friend_account.Identity()
		friend_identity, err := friend.Identity()
		assert.NoError(t, err)
		assert.Equal(t, friend_account_identity, friend_identity)
	})

	t.Run("File should be encrypted for this account", func(t T) {
		anotherDir := t.TempDir()
		anotherAccount, _ := NewAccount(anotherDir, "Unknown account", "unknow@example.com", "password")

		file_content, _ := os.ReadFile(path.Join(dir, ".mau", fingerprint.String()+".pgp"))
		if err := os.WriteFile(path.Join(anotherDir, ".mau", fingerprint.String()+".pgp"), file_content, DirPerm); err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		friends, err := anotherAccount.ListFriends()
		assert.Error(t, err, "ListFriends should fail to decrypt a friend")
		assert.Nil(t, friends)
	})
}

func TestRemoveFriend(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	var friend_account_pub bytes.Buffer
	if err := friend_account.Export(&friend_account_pub); err != nil {
		t.Fatalf("Failed to export friend account: %v", err)
	}
	fingerprint := friend_account.Fingerprint()
	friend, _ := account.AddFriend(&friend_account_pub)

	err := account.RemoveFriend(friend)
	assert.NoError(t, err)
	assert.NoFileExists(t, path.Join(dir, ".mau", fingerprint.String()+".pgp"))
}

func TestListFriends(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	var friend_account_pub bytes.Buffer
	if err := friend_account.Export(&friend_account_pub); err != nil {
		t.Fatalf("Failed to export friend account: %v", err)
	}
	friend, _ := account.AddFriend(&friend_account_pub)

	keyring, err := account.ListFriends()
	assert.NoError(t, err)
	assert.NotEqual(t, nil, keyring)
	assert.Equal(t, path.Join(dir, ".mau"), keyring.Path)
	assert.Equal(t, 1, len(keyring.Friends))
	assert.Equal(t, 0, len(keyring.SubKeyrings))

	err = account.RemoveFriend(friend)
	assert.NoError(t, err)

	keyring, err = account.ListFriends()
	assert.NoError(t, err)
	assert.NotEqual(t, nil, keyring)
	assert.Equal(t, path.Join(dir, ".mau"), keyring.Path)
	assert.Equal(t, 0, len(keyring.Friends))
	assert.Equal(t, 0, len(keyring.SubKeyrings))
}
