package mau

import (
	"bytes"
	"context"
	"log"
	"net"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	account, err := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "strong password")
	assert.NoError(t, err)

	client, err := account.Client(account.Fingerprint(), nil)
	assert.NoError(t, err)
	assert.NotEqual(t, nil, client)
}

func TestDownloadFriend(t *testing.T) {
	account_dir := t.TempDir()
	account, err := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")
	assert.NoError(t, err)
	var account_key bytes.Buffer
	err = account.Export(&account_key)
	assert.NoError(t, err)

	friend_dir := t.TempDir()
	friend, err := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	assert.NoError(t, err)
	var friend_key bytes.Buffer
	if err := friend.Export(&friend_key); err != nil {
		t.Fatalf("Failed to export friend key: %v", err)
	}
	server, _ := friend.Server(nil)

	listener, address := TempListener()
	go func() {
		if err := server.Serve(*listener, ""); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()
	defer server.Close()

	client, _ := account.Client(friend.Fingerprint(), nil)

	t.Run("When the fingerprint is not a friend", func(t T) {
		err := client.DownloadFriend(context.Background(), friend.Fingerprint(), time.Now(), []FingerprintResolver{StaticAddress(address)})
		assert.Error(t, ErrFriendNotFollowed, err)
	})

	t.Run("When friend but not followed", func(t T) {
		if _, err := account.AddFriend(bytes.NewReader(friend_key.Bytes())); err != nil {
			t.Fatalf("Failed to add friend: %v", err)
		}
		err := client.DownloadFriend(context.Background(), friend.Fingerprint(), time.Now(), []FingerprintResolver{StaticAddress(address)})
		assert.Error(t, ErrFriendNotFollowed, err)
	})

	t.Run("When friend and followed", func(t T) {
		f, err := account.AddFriend(bytes.NewReader(friend_key.Bytes()))
		assert.NoError(t, err)
		if err := account.Follow(f); err != nil {
			t.Fatalf("Failed to follow friend: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = client.DownloadFriend(ctx, friend.Fingerprint(), time.Now(), []FingerprintResolver{StaticAddress(address)})
		assert.NoError(t, err)
	})

	t.Run("When a file is encrypted for friend", func(t T) {
		// Create a file in the friend account
		aFriend, _ := friend.AddFriend(bytes.NewReader(account_key.Bytes()))
		_, err := friend.AddFile(strings.NewReader("Hello world!"), "hello world.txt", []*Friend{aFriend})
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(friend_dir, friend.Fingerprint().String(), "hello world.txt.pgp"))

		// and download it to the account
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = client.DownloadFriend(ctx, friend.Fingerprint(), time.Now().Add(-time.Second), []FingerprintResolver{StaticAddress(address)})
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(account_dir, friend.Fingerprint().String(), "hello world.txt.pgp"))
	})

	t.Run("When private file exists", func(t T) {
		_, err := friend.AddFile(strings.NewReader("Private social security number"), "private.txt", []*Friend{})
		assert.NoError(t, err)
		assert.FileExists(t, path.Join(friend_dir, friend.Fingerprint().String(), "private.txt.pgp"))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		err = client.DownloadFriend(ctx, friend.Fingerprint(), time.Now().Add(-time.Second), []FingerprintResolver{StaticAddress(address)})
		assert.NoError(t, err)
		assert.NoFileExists(t, path.Join(account_dir, friend.Fingerprint().String(), "private.txt.pgp"))
	})

	t.Run("When no address is provided it find the user on the local network", func(t T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DownloadFriend(ctx, friend.Fingerprint(), time.Now().Add(-time.Second), []FingerprintResolver{LocalFriendAddress})
		assert.NoError(t, err)
	})

	t.Run("Connecting to an account with wrong peer ID", func(t T) {
		anotherFriend, _ := NewAccount(t.TempDir(), "Another person", "another@example.com", "password")
		client, _ := account.Client(anotherFriend.Fingerprint(), nil)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := client.DownloadFriend(ctx, friend.Fingerprint(), time.Now().Add(-time.Second), []FingerprintResolver{StaticAddress(address)})
		assert.Error(t, ErrIncorrectPeerCertificate, err)
	})
}

func Timeout(p time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), p)
	_ = cancel // Suppress lostcancel warning - caller should manage context
	return ctx
}

func TempListener() (*net.Listener, string) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		log.Fatal("Error while creating listener for testing:", err.Error())
	}

	address := listener.Addr().(*net.TCPAddr).String()
	url := address
	return &listener, url
}
