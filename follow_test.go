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
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")
	assert.NoError(t, err)

	friend_dir := t.TempDir()
	friend_account, err := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	assert.NoError(t, err)
	var friend_pub bytes.Buffer
	err = friend_account.Export(&friend_pub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friend_pub)
	assert.NoError(t, err)

	err = account.Follow(friend)
	assert.NoError(t, err)
	assert.DirExists(t, path.Join(dir, friend_account.Fingerprint().String()))
}

func TestUnfollow(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")
	assert.NoError(t, err)

	friend_dir := t.TempDir()
	friend_account, err := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	assert.NoError(t, err)
	var friend_pub bytes.Buffer
	err = friend_account.Export(&friend_pub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friend_pub)
	assert.NoError(t, err)

	err = account.Follow(friend)
	assert.NoError(t, err)
	err = account.Unfollow(friend)
	assert.NoError(t, err)
	assert.DirExists(t, path.Join(dir, "."+friend_account.Fingerprint().String()))
}

func TestListFollows(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")
	assert.NoError(t, err)

	friend_dir := t.TempDir()
	friend_account, err := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	assert.NoError(t, err)
	var friend_pub bytes.Buffer
	err = friend_account.Export(&friend_pub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friend_pub)
	assert.NoError(t, err)

	t.Run("Before following anyone", func(t T) {
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(follows))
	})

	t.Run("After following a friend", func(t T) {
		err := account.Follow(friend)
		assert.NoError(t, err)
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(follows))
	})

	t.Run("After unfollowing the friend", func(t T) {
		err := account.Unfollow(friend)
		assert.NoError(t, err)
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(follows))
	})

	t.Run("When having a dir that's not a fingerprint", func(t T) {
		err := os.Mkdir(path.Join(account.path, "systemdir"), 0777)
		assert.NoError(t, err)
		follows, err := account.ListFollows()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(follows))
	})

}
