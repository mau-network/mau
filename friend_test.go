package main

import (
	"bytes"
	"os"
	"path"
	"testing"
)

func TestAddFriend(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	fingerprint := friend_account.Fingerprint()
	friend_account_pub, _ := friend_account.Export()
	friend, err := AddFriend(account, bytes.NewBuffer(friend_account_pub))
	ASSERT_ERROR(t, nil, err)
	ASSERT_FILE_EXISTS(t, path.Join(dir, ".mau", fingerprint.String()+".pgp"))

	t.Run("Email", func(t T) {
		ASSERT_EQUAL(t, "mohamed@example.com", friend.Email())
	})

	t.Run("Name", func(t T) {
		ASSERT_EQUAL(t, "Mohamed Mahmoud", friend.Name())
	})

	t.Run("Fingerprint", func(t T) {
		ASSERT_EQUAL(t, friend_account.Fingerprint(), friend.Fingerprint())
	})

	t.Run("Identity", func(t T) {
		friend_account_identity, _ := friend_account.Identity()
		friend_identity, err := friend.Identity()
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, friend_account_identity, friend_identity)
	})

	t.Run("File should be encrypted for this account", func(t T) {
		anotherDir := t.TempDir()
		anotherAccount, _ := NewAccount(anotherDir, "Unknown account", "unknow@example.com", "password")

		file_content, _ := os.ReadFile(path.Join(dir, ".mau", fingerprint.String()+".pgp"))
		os.WriteFile(path.Join(anotherDir, ".mau", fingerprint.String()+".pgp"), file_content, 0700)

		friends, err := ListFriends(anotherAccount)
		ASSERT(t, err != nil, "ListFriends should fail to decrypt a friend")
		ASSERT_EQUAL(t, nil, friends)
	})
}

func TestRemoveFriend(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_account_pub, _ := friend_account.Export()
	fingerprint := friend_account.Fingerprint()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_account_pub))

	err := RemoveFriend(account, friend)
	ASSERT_ERROR(t, nil, err)
	REFUTE_FILE_EXISTS(t, path.Join(dir, ".mau", fingerprint.String()+".pgp"))
}

func TestListFriends(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_account_pub, _ := friend_account.Export()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_account_pub))

	keyring, err := ListFriends(account)
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, keyring)
	ASSERT_EQUAL(t, path.Join(dir, ".mau"), keyring.Path)
	ASSERT_EQUAL(t, 1, len(keyring.Friends))
	ASSERT_EQUAL(t, 0, len(keyring.KeyRings))

	err = RemoveFriend(account, friend)

	keyring, err = ListFriends(account)
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, keyring)
	ASSERT_EQUAL(t, path.Join(dir, ".mau"), keyring.Path)
	ASSERT_EQUAL(t, 0, len(keyring.Friends))
	ASSERT_EQUAL(t, 0, len(keyring.KeyRings))
}
