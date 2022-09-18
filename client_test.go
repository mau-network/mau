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
)

func TestNewClient(t *testing.T) {
	account, err := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "strong password")

	client, err := NewClient(account)
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, client)
}

func TestDownloadFriend(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")
	account_key, _ := account.Export()
	client, _ := NewClient(account)

	friend_dir := t.TempDir()
	friend, _ := NewAccount(friend_dir, "Mohamed Mahmoud", "mohamed@example.com", "strong password")
	friend_key, _ := friend.Export()
	server, _ := NewServer(friend)

	listener, address := TempListener()
	go server.Serve(*listener)

	t.Run("When the fingerprint is not a friend", func(t T) {
		err := DownloadFriend(context.Background(), account, address, friend.Fingerprint(), time.Now(), client)
		ASSERT_ERROR(t, ErrFriendNotFollowed, err)
	})

	t.Run("When friend but not followed", func(t T) {
		AddFriend(account, bytes.NewBuffer(friend_key))
		err := DownloadFriend(context.Background(), account, address, friend.Fingerprint(), time.Now(), client)
		ASSERT_ERROR(t, ErrFriendNotFollowed, err)
	})

	t.Run("When friend and followed", func(t T) {
		f, _ := AddFriend(account, bytes.NewBuffer(friend_key))
		Follow(account, f)

		err := DownloadFriend(Timeout(time.Second), account, address, friend.Fingerprint(), time.Now(), client)
		ASSERT_ERROR(t, nil, err)
	})

	t.Run("When a file is encrypted for friend", func(t T) {
		// Create a file in the friend account
		aFriend, _ := AddFriend(friend, bytes.NewBuffer(account_key))
		_, err := AddFile(friend, strings.NewReader("Hello world!"), "hello world.txt", []*Friend{aFriend})
		ASSERT_ERROR(t, nil, err)
		ASSERT_FILE_EXISTS(t, path.Join(friend_dir, friend.Fingerprint().String(), "hello world.txt.pgp"))

		// and download it to the account
		err = DownloadFriend(Timeout(time.Second), account, address, friend.Fingerprint(), time.Now().Add(-time.Second), client)
		ASSERT_ERROR(t, nil, err)
		ASSERT_FILE_EXISTS(t, path.Join(account_dir, friend.Fingerprint().String(), "hello world.txt.pgp"))
	})

	t.Run("When private file exists", func(t T) {
		_, err := AddFile(friend, strings.NewReader("Private social security number"), "private.txt", []*Friend{})
		ASSERT_ERROR(t, nil, err)
		ASSERT_FILE_EXISTS(t, path.Join(friend_dir, friend.Fingerprint().String(), "private.txt.pgp"))

		err = DownloadFriend(Timeout(time.Second), account, address, friend.Fingerprint(), time.Now().Add(-time.Second), client)
		ASSERT_ERROR(t, nil, err)
		REFUTE_FILE_EXISTS(t, path.Join(account_dir, friend.Fingerprint().String(), "private.txt.pgp"))
	})

	t.Run("When no address is provided it find the user on the local network", func(t T) {
		ctx := Timeout(10 * time.Second)
		err := DownloadFriend(ctx, account, "", friend.Fingerprint(), time.Now().Add(-time.Second), client)
		ASSERT_ERROR(t, nil, err)
	})
}

func Timeout(p time.Duration) context.Context {
	ctx, _ := context.WithTimeout(context.Background(), p)
	return ctx
}

func TempListener() (*net.Listener, string) {
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		log.Fatal("Error while creating listener for testing:", err.Error())
	}

	address := listener.Addr().(*net.TCPAddr).String()
	url := "https://" + address
	return &listener, url
}
