package main

import (
	"bytes"
	"testing"
)

func TestFriend(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_account_pub, _ := friend_account.Export()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_account_pub))

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
}
