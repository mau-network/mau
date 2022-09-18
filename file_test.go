package mau

import (
	"encoding/hex"
	"io"
	"path"
	"strings"
	"testing"
	"time"
)

func TestFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, err := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	ASSERT_ERROR(t, nil, err)
	ASSERT_EQUAL(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp"), file.Path)

	versions := file.Versions()
	ASSERT_EQUAL(t, 0, len(versions))

	ASSERT_EQUAL(t, "hello.txt.pgp", file.Name())
	ASSERT_EQUAL(t, false, file.Deleted(account))

	size, err := file.Size()
	ASSERT_ERROR(t, nil, err)
	ASSERT(t, size > 0, "Size should not be zero")

	hash, err := file.Hash()
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, "", hash)

	reader, err := file.Reader(account)
	ASSERT_ERROR(t, nil, err)

	content, err := io.ReadAll(reader)
	ASSERT_ERROR(t, nil, err)

	ASSERT_EQUAL(t, "hello world", string(content))

	friends, err := file.Recipients(account)
	ASSERT_ERROR(t, nil, err)
	ASSERT_EQUAL(t, 0, len(friends))

	t.Run("Versions", func(t T) {
		file, err := account.AddFile(strings.NewReader("hello there"), "hello.txt", []*Friend{})
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp"), file.Path)

		versions := file.Versions()
		ASSERT_EQUAL(t, 1, len(versions))
		ASSERT_DIR_EXISTS(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp.versions"))

		version := versions[0]
		ASSERT_FILE_EXISTS(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp.versions", version.Name()))

		reader, err := version.Reader(account)
		content, err := io.ReadAll(reader)
		ASSERT_EQUAL(t, "hello world", string(content))

		nameInbytes, err := hex.DecodeString(version.Name())
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, 32, len(nameInbytes))
	})
}

func TestGetFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	opened, _ := account.GetFile(account.Fingerprint(), "hello.txt.pgp")

	ASSERT_EQUAL(t, file.Path, opened.Path)
	ASSERT_EQUAL(t, file.Name(), opened.Name())
	ASSERT_EQUAL(t, file.Deleted(account), opened.Deleted(account))
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
	ASSERT(t, !file.Deleted(account), "File should exist (not deleted)")

	err := account.RemoveFile(file)
	ASSERT_ERROR(t, nil, err)

	ASSERT(t, file.Deleted(account), "File should be deleted")

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
		unknownFpr, _ := ParseFingerprint("01234567891234567890")
		files := account.ListFiles(unknownFpr, time.Now().Add(-time.Second), 10)
		ASSERT_EQUAL(t, 0, len(files))
	})
}
