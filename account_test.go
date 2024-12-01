package mau

import (
	"bytes"
	"io"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewAccount(t *testing.T) {
	t.Run("Creating an account with valid parameters", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

		assert.NoError(t, err, "Error was returned when creating an account: %s", err)
		assert.NotNil(t, account, "Account value is nil, expected a value")

		t.Run("Include correct information", func(t T) {
			identity, _ := account.Identity()
			var pgpkey bytes.Buffer
			account.Export(&pgpkey)

			assert.Equal(t, "ahmed@example.com", account.Email())
			assert.Equal(t, "Ahmed Mohamed", account.Name())
			assert.Equal(t, "Ahmed Mohamed <ahmed@example.com>", identity)
			assert.NotEqual(t, 0, len(pgpkey.Bytes()))
		})

		t.Run("Creates the correct file structure", func(t T) {
			assert.DirExists(t, path.Join(dir, ".mau"))
			assert.FileExists(t, path.Join(dir, ".mau", accountKeyFilename))
		})
	})

	t.Run("Creating an account without a password", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "")

		assert.Error(t, ErrPassphraseRequired, err)
		assert.Nil(t, account)
	})

	t.Run("Creating an account in an existing account directory", func(t T) {
		dir := t.TempDir()
		NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "password")
		account, err := NewAccount(dir, "Ahmed Mahmoud", "ahmed.mahmoud@example.com", "password")

		assert.ErrorIs(t, err, ErrAccountAlreadyExists, "Expected an error: %s Got: %s", ErrAccountAlreadyExists, err)
		assert.Nil(t, account, "Expected the account to be nil value got : %v", account)
	})

	t.Run("Two accounts with same identity", func(t T) {
		account1, _ := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "password")
		account2, _ := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "password")

		assert.NotEqual(t, account1.Fingerprint(), account2.Fingerprint())
	})
}

func TestOpenAccount(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	t.Run("Using same password", func(t T) {
		opened, err := OpenAccount(dir, "strong password")
		assert.NoError(t, err)
		assert.Equal(t, "ahmed@example.com", opened.Email())
		assert.Equal(t, "Ahmed Mohamed", opened.Name())
		assert.Equal(t, account.Fingerprint(), opened.Fingerprint())
	})

	t.Run("Using wrong password", func(t T) {
		opened, err := OpenAccount(dir, "wrong password")
		assert.Error(t, ErrIncorrectPassphrase, err)
		assert.Nil(t, opened)
	})
}

func TestGetFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	opened, _ := account.GetFile(account.Fingerprint(), "hello.txt.pgp")

	assert.Equal(t, file.Path, opened.Path)
	assert.Equal(t, file.Name(), opened.Name())
	assert.Equal(t, file.Deleted(), opened.Deleted())
	assert.Equal(t, len(file.Versions()), len(opened.Versions()))

	file_hash, _ := file.Hash()
	opened_hash, _ := opened.Hash()
	assert.Equal(t, file_hash, opened_hash)

	file_reader, _ := file.Reader(account)
	opened_reader, _ := opened.Reader(account)

	file_content, _ := io.ReadAll(file_reader)
	opened_content, _ := io.ReadAll(opened_reader)
	assert.Equal(t, string(file_content), string(opened_content))
}

func TestRemoveFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	assert.False(t, file.Deleted(), "File should exist (not deleted)")

	file, _ = account.AddFile(strings.NewReader("bye world"), "hello.txt", []*Friend{})
	assert.Equal(t, 1, len(file.Versions()))

	err := account.RemoveFile(file)
	assert.NoError(t, err)

	assert.True(t, file.Deleted(), "File should be deleted")
	assert.Equal(t, 0, len(file.Versions()))

	recipients, err := file.Recipients(account)
	assert.Equal(t, 0, len(recipients))
}

func TestListFiles(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})

	t.Run("Asking for 1 second old files", func(t T) {
		files := account.ListFiles(account.Fingerprint(), time.Now().Add(-time.Second), 10)
		assert.Equal(t, 1, len(files))
	})

	t.Run("Asking for 0 seconds old files", func(t T) {
		files := account.ListFiles(account.Fingerprint(), time.Now().Add(time.Second), 10)
		assert.Equal(t, 0, len(files))
	})

	t.Run("Asking for a fingerprint other than the account", func(t T) {
		unknownFpr, _ := FingerprintFromString("01234567891234567890")
		files := account.ListFiles(unknownFpr, time.Now().Add(-time.Second), 10)
		assert.Equal(t, 0, len(files))
	})
}
