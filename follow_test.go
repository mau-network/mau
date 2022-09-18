package mau

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
	var friend_pub bytes.Buffer
	friend_account.Export(&friend_pub)
	friend, _ := account.AddFriend(&friend_pub)

	err := account.Follow(friend)
	ASSERT_NO_ERROR(t, err)
	ASSERT_DIR_EXISTS(t, path.Join(dir, friend_account.Fingerprint().String()))
}

func TestUnfollow(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	var friend_pub bytes.Buffer
	friend_account.Export(&friend_pub)
	friend, _ := account.AddFriend(&friend_pub)

	account.Follow(friend)
	err := account.Unfollow(friend)
	ASSERT_NO_ERROR(t, err)
	ASSERT_DIR_EXISTS(t, path.Join(dir, "."+friend_account.Fingerprint().String()))
}

func TestListFollows(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	friend_dir := t.TempDir()
	friend_account, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	var friend_pub bytes.Buffer
	friend_account.Export(&friend_pub)
	friend, _ := account.AddFriend(&friend_pub)

	t.Run("Before following anyone", func(t T) {
		follows, _ := account.ListFollows()
		ASSERT_EQUAL(t, 0, len(follows))
	})

	t.Run("After following a friend", func(t T) {
		account.Follow(friend)
		follows, err := account.ListFollows()
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, 1, len(follows))
	})

	t.Run("After unfollowing the friend", func(t T) {
		account.Unfollow(friend)
		follows, _ := account.ListFollows()
		ASSERT_EQUAL(t, 0, len(follows))
	})

}
