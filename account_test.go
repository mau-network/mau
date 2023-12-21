package mau

import (
	"bytes"
	"io"
	"path"
	"strings"
	"testing"
	"time"
)

func TestNewAccount(t *testing.T) {
	t.Run("Creating an account with valid parameters", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

		ASSERT(t, err == nil, "Error was returned when creating an account: %s", err)
		ASSERT(t, account != nil, "Account value is nil, expected a value")

		t.Run("Include correct information", func(t T) {
			identity, _ := account.Identity()
			var pgpkey bytes.Buffer
			account.Export(&pgpkey)

			ASSERT_EQUAL(t, "ahmed@example.com", account.Email())
			ASSERT_EQUAL(t, "Ahmed Mohamed", account.Name())
			ASSERT_EQUAL(t, "Ahmed Mohamed <ahmed@example.com>", identity)
			REFUTE_EQUAL(t, 0, len(pgpkey.Bytes()))
		})

		t.Run("Creates the correct file structure", func(t T) {
			ASSERT_DIR_EXISTS(t, path.Join(dir, ".mau"))
			ASSERT_FILE_EXISTS(t, path.Join(dir, ".mau", accountKeyFilename))
		})
	})

	t.Run("Creating an account without a password", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "")

		ASSERT_ERROR(t, ErrPassphraseRequired, err)
		ASSERT_EQUAL(t, nil, account)
	})

	t.Run("Creating an account in an existing account directory", func(t T) {
		dir := t.TempDir()
		NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "password")
		account, err := NewAccount(dir, "Ahmed Mahmoud", "ahmed.mahmoud@example.com", "password")

		ASSERT(t, err == ErrAccountAlreadyExists, "Expected an error: %s Got: %s", ErrAccountAlreadyExists, err)
		ASSERT(t, account == nil, "Expected the account to be nil value got : %v", account)
	})

	t.Run("Two accounts with same identity", func(t T) {
		account1, _ := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "password")
		account2, _ := NewAccount(t.TempDir(), "Ahmed Mohamed", "ahmed@example.com", "password")

		REFUTE_EQUAL(t, account1.Fingerprint(), account2.Fingerprint())
	})
}

func TestOpenAccount(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "strong password")

	t.Run("Using same password", func(t T) {
		opened, err := OpenAccount(dir, "strong password")
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, "ahmed@example.com", opened.Email())
		ASSERT_EQUAL(t, "Ahmed Mohamed", opened.Name())
		ASSERT_EQUAL(t, account.Fingerprint(), opened.Fingerprint())
	})

	t.Run("Using wrong password", func(t T) {
		opened, err := OpenAccount(dir, "wrong password")
		ASSERT_ERROR(t, ErrIncorrectPassphrase, err)
		ASSERT_EQUAL(t, nil, opened)
	})
}

func TestGetFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	opened, _ := account.GetFile(account.Fingerprint(), "hello.txt.pgp")

	ASSERT_EQUAL(t, file.Path, opened.Path)
	ASSERT_EQUAL(t, file.Name(), opened.Name())
	ASSERT_EQUAL(t, file.Deleted(), opened.Deleted())
	ASSERT_EQUAL(t, len(file.Versions()), len(opened.Versions()))

	file_hash, _ := file.Hash()
	opened_hash, _ := opened.Hash()
	ASSERT_EQUAL(t, file_hash, opened_hash)

	file_reader, _ := file.Reader(account)
	opened_reader, _ := opened.Reader(account)

	file_content, _ := io.ReadAll(file_reader)
	opened_content, _ := io.ReadAll(opened_reader)
	ASSERT_EQUAL(t, string(file_content), string(opened_content))
}

func TestRemoveFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	ASSERT(t, !file.Deleted(), "File should exist (not deleted)")

	file, _ = account.AddFile(strings.NewReader("bye world"), "hello.txt", []*Friend{})
	ASSERT_EQUAL(t, 1, len(file.Versions()))

	err := account.RemoveFile(file)
	ASSERT_NO_ERROR(t, err)

	ASSERT(t, file.Deleted(), "File should be deleted")
	ASSERT_EQUAL(t, 0, len(file.Versions()))

	recipients, err := file.Recipients(account)
	ASSERT_EQUAL(t, 0, len(recipients))
}

func TestListFiles(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})

	t.Run("Asking for 1 second old files", func(t T) {
		files := account.ListFiles(account.Fingerprint(), time.Now().Add(-time.Second), 10)
		ASSERT_EQUAL(t, 1, len(files))
	})

	t.Run("Asking for 0 seconds old files", func(t T) {
		files := account.ListFiles(account.Fingerprint(), time.Now().Add(time.Second), 10)
		ASSERT_EQUAL(t, 0, len(files))
	})

	t.Run("Asking for a fingerprint other than the account", func(t T) {
		unknownFpr, _ := FingerprintFromString("01234567891234567890")
		files := account.ListFiles(unknownFpr, time.Now().Add(-time.Second), 10)
		ASSERT_EQUAL(t, 0, len(files))
	})
}
