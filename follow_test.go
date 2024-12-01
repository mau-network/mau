package mau

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err)
	assert.DirExists(t, path.Join(dir, friend_account.Fingerprint().String()))
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
	assert.NoError(t, err)
	assert.DirExists(t, path.Join(dir, "."+friend_account.Fingerprint().String()))
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
		assert.Equal(t, 0, len(follows))
	})

	t.Run("After following a friend", func(t T) {
		account.Follow(friend)
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(follows))
	})

	t.Run("After unfollowing the friend", func(t T) {
		account.Unfollow(friend)
		follows, _ := account.ListFollows()
		assert.Equal(t, 0, len(follows))
	})

	t.Run("When having a dir that's not a fingerprint", func(t T) {
		os.Mkdir(path.Join(account.path, "systemdir"), 0777)
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(follows))
	})

}
