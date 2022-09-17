package main

import (
	"bytes"
	"path"
	"testing"
)

func TestFollow(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_pub, _ := friend_account.Export()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_pub))

	err := Follow(account, friend)
	ASSERT_ERROR(t, nil, err)
	ASSERT_DIR_EXISTS(t, path.Join(dir, friend_account.Fingerprint().String()))
}

func TestUnfollow(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_pub, _ := friend_account.Export()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_pub))

	Follow(account, friend)
	err := Unfollow(account, friend)
	ASSERT_ERROR(t, nil, err)
	ASSERT_DIR_EXISTS(t, path.Join(dir, "."+friend_account.Fingerprint().String()))
}

func TestListFollows(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_pub, _ := friend_account.Export()
	friend, _ := AddFriend(account, bytes.NewBuffer(friend_pub))

	t.Run("Before following anyone", func(t T) {
		follows, _ := ListFollows(account)
		ASSERT_EQUAL(t, 0, len(follows))
	})

	t.Run("After following a friend", func(t T) {
		Follow(account, friend)
		follows, _ := ListFollows(account)
		ASSERT_EQUAL(t, 1, len(follows))
	})

	t.Run("After unfollowing the friend", func(t T) {
		Unfollow(account, friend)
		follows, _ := ListFollows(account)
		ASSERT_EQUAL(t, 0, len(follows))
	})

}
