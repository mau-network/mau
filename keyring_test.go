package mau

import (
	"bytes"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyring_Name(t *testing.T) {
	tests := []struct {
		name     string
		keyring  *Keyring
		expected string
	}{
		{
			name:     "returns base name of path",
			keyring:  &Keyring{Path: "/home/user/.mau/work"},
			expected: "work",
		},
		{
			name:     "returns base name for nested path",
			keyring:  &Keyring{Path: "/home/user/.mau/work/colleagues"},
			expected: "colleagues",
		},
		{
			name:     "returns empty string for nil keyring",
			keyring:  nil,
			expected: "",
		},
		{
			name:     "returns empty string for empty path",
			keyring:  &Keyring{Path: ""},
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.keyring.Name()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeyring_FriendsSet(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Main User", "main@example.com", "password")
	require.NoError(t, err)

	// Create test friends
	friend1Dir := t.TempDir()
	friend1Account, err := NewAccount(friend1Dir, "Friend One", "friend1@example.com", "password")
	require.NoError(t, err)
	var friend1Pub bytes.Buffer
	err = friend1Account.Export(&friend1Pub)
	require.NoError(t, err)
	friend1, err := account.AddFriend(&friend1Pub)
	require.NoError(t, err)

	friend2Dir := t.TempDir()
	friend2Account, err := NewAccount(friend2Dir, "Friend Two", "friend2@example.com", "password")
	require.NoError(t, err)
	var friend2Pub bytes.Buffer
	err = friend2Account.Export(&friend2Pub)
	require.NoError(t, err)
	friend2, err := account.AddFriend(&friend2Pub)
	require.NoError(t, err)

	// Create sub-keyring directory and add friend to it
	friend3Dir := t.TempDir()
	friend3Account, err := NewAccount(friend3Dir, "Friend Three", "friend3@example.com", "password")
	require.NoError(t, err)
	var friend3Pub bytes.Buffer
	err = friend3Account.Export(&friend3Pub)
	require.NoError(t, err)
	friend3, err := account.AddFriend(&friend3Pub)
	require.NoError(t, err)

	// Move friend3 to sub-keyring directory
	subKeyringPath := path.Join(dir, ".mau", "work")
	err = os.MkdirAll(subKeyringPath, DirPerm)
	require.NoError(t, err)

	oldPath := path.Join(dir, ".mau", friend3.Fingerprint().String()+".pgp")
	newPath := path.Join(subKeyringPath, friend3.Fingerprint().String()+".pgp")
	err = os.Rename(oldPath, newPath)
	require.NoError(t, err)

	// Load keyring
	keyring, err := account.ListFriends()
	require.NoError(t, err)

	t.Run("returns all friends including from sub-keyrings", func(t *testing.T) {
		friendsSet := keyring.FriendsSet()
		assert.Len(t, friendsSet, 3)

		// Verify all friends are present by fingerprint
		fingerprints := make(map[Fingerprint]bool)
		for _, f := range friendsSet {
			fingerprints[f.Fingerprint()] = true
		}

		assert.True(t, fingerprints[friend1.Fingerprint()])
		assert.True(t, fingerprints[friend2.Fingerprint()])
		assert.True(t, fingerprints[friend3.Fingerprint()])
	})

	t.Run("deduplicates friends across keyrings", func(t *testing.T) {
		// Copy friend1 file to sub-keyring as well
		friend1SrcFile := path.Join(dir, ".mau", friend1.Fingerprint().String()+".pgp")
		friend1DstFile := path.Join(subKeyringPath, friend1.Fingerprint().String()+".pgp")
		
		data, err := os.ReadFile(friend1SrcFile)
		require.NoError(t, err)
		err = os.WriteFile(friend1DstFile, data, FilePerm)
		require.NoError(t, err)

		// Reload keyring
		keyring, err := account.ListFriends()
		require.NoError(t, err)

		friendsSet := keyring.FriendsSet()
		// Should still be 3 unique friends (friend1 appears in both keyrings but counted once)
		assert.Len(t, friendsSet, 3)
	})
}

func TestKeyring_FindByFingerprint(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Main User", "main@example.com", "password")
	require.NoError(t, err)

	// Create test friends
	friend1Dir := t.TempDir()
	friend1Account, err := NewAccount(friend1Dir, "Friend One", "friend1@example.com", "password")
	require.NoError(t, err)
	var friend1Pub bytes.Buffer
	err = friend1Account.Export(&friend1Pub)
	require.NoError(t, err)
	friend1, err := account.AddFriend(&friend1Pub)
	require.NoError(t, err)

	// Create sub-keyring with another friend
	friend2Dir := t.TempDir()
	friend2Account, err := NewAccount(friend2Dir, "Friend Two", "friend2@example.com", "password")
	require.NoError(t, err)
	var friend2Pub bytes.Buffer
	err = friend2Account.Export(&friend2Pub)
	require.NoError(t, err)
	friend2, err := account.AddFriend(&friend2Pub)
	require.NoError(t, err)

	// Move friend2 to sub-keyring
	subKeyringPath := path.Join(dir, ".mau", "work")
	err = os.MkdirAll(subKeyringPath, DirPerm)
	require.NoError(t, err)
	
	oldPath := path.Join(dir, ".mau", friend2.Fingerprint().String()+".pgp")
	newPath := path.Join(subKeyringPath, friend2.Fingerprint().String()+".pgp")
	err = os.Rename(oldPath, newPath)
	require.NoError(t, err)

	keyring, err := account.ListFriends()
	require.NoError(t, err)

	t.Run("finds friend in root keyring", func(t *testing.T) {
		found := keyring.FindByFingerprint(friend1.Fingerprint())
		assert.NotNil(t, found)
		assert.Equal(t, friend1.Fingerprint(), found.Fingerprint())
		assert.Equal(t, "Friend One", found.Name())
	})

	t.Run("finds friend in sub-keyring", func(t *testing.T) {
		found := keyring.FindByFingerprint(friend2.Fingerprint())
		assert.NotNil(t, found)
		assert.Equal(t, friend2.Fingerprint(), found.Fingerprint())
		assert.Equal(t, "Friend Two", found.Name())
	})

	t.Run("returns nil for non-existent fingerprint", func(t *testing.T) {
		// Create a properly formatted but non-existent fingerprint
		nonExistentFP := Fingerprint([20]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		found := keyring.FindByFingerprint(nonExistentFP)
		assert.Nil(t, found)
	})

	t.Run("returns nil for nil keyring", func(t *testing.T) {
		var nilKeyring *Keyring
		found := nilKeyring.FindByFingerprint(friend1.Fingerprint())
		assert.Nil(t, found)
	})
}

func TestKeyring_FriendById(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Main User", "main@example.com", "password")
	require.NoError(t, err)

	// Create test friend
	friendDir := t.TempDir()
	friendAccount, err := NewAccount(friendDir, "Test Friend", "friend@example.com", "password")
	require.NoError(t, err)
	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	require.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	require.NoError(t, err)

	keyring, err := account.ListFriends()
	require.NoError(t, err)

	t.Run("finds friend by primary key ID", func(t *testing.T) {
		keyId := friend.entity.PrimaryKey.KeyId
		found := keyring.FriendById(keyId)
		assert.NotNil(t, found)
		assert.Equal(t, friend.Fingerprint(), found.Fingerprint())
	})

	t.Run("finds friend by subkey ID", func(t *testing.T) {
		if len(friend.entity.Subkeys) > 0 {
			subkeyId := friend.entity.Subkeys[0].PublicKey.KeyId
			found := keyring.FriendById(subkeyId)
			assert.NotNil(t, found)
			assert.Equal(t, friend.Fingerprint(), found.Fingerprint())
		}
	})

	t.Run("returns nil for non-existent key ID", func(t *testing.T) {
		nonExistentId := uint64(0xDEADBEEF)
		found := keyring.FriendById(nonExistentId)
		assert.Nil(t, found)
	})
}

func TestKeyring_FriendById_SubKeyrings(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Main User", "main@example.com", "password")
	require.NoError(t, err)

	// Create a friend and move to sub-keyring
	friendDir := t.TempDir()
	friendAccount, err := NewAccount(friendDir, "Work Friend", "work@example.com", "password")
	require.NoError(t, err)
	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	require.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	require.NoError(t, err)

	// Move to sub-keyring
	subKeyringPath := path.Join(dir, ".mau", "work")
	err = os.MkdirAll(subKeyringPath, DirPerm)
	require.NoError(t, err)
	
	oldPath := path.Join(dir, ".mau", friend.Fingerprint().String()+".pgp")
	newPath := path.Join(subKeyringPath, friend.Fingerprint().String()+".pgp")
	err = os.Rename(oldPath, newPath)
	require.NoError(t, err)

	keyring, err := account.ListFriends()
	require.NoError(t, err)

	t.Run("finds friend in sub-keyring by key ID", func(t *testing.T) {
		keyId := friend.entity.PrimaryKey.KeyId
		found := keyring.FriendById(keyId)
		assert.NotNil(t, found)
		assert.Equal(t, friend.Fingerprint(), found.Fingerprint())
		assert.Equal(t, "Work Friend", found.Name())
	})
}
