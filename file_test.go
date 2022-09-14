package main

import (
	"io"
	"path"
	"strings"
	"testing"
	"time"
)

func TestFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, err := AddFile(account, strings.NewReader("hello world"), "hello.txt", []*Friend{})
	ASSERT_ERROR(t, nil, err)
	ASSERT_EQUAL(t, path.Join(account_dir, account.Fingerprint(), "hello.txt.pgp"), file.Path)

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
}

func TestGetFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, _ := AddFile(account, strings.NewReader("hello world"), "hello.txt", []*Friend{})
	opened, _ := GetFile(account, account.Fingerprint(), "hello.txt.pgp")

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

	file, _ := AddFile(account, strings.NewReader("hello world"), "hello.txt", []*Friend{})
	ASSERT(t, !file.Deleted(account), "File should exist (not deleted)")

	err := RemoveFile(account, file)
	ASSERT_ERROR(t, nil, err)

	ASSERT(t, file.Deleted(account), "File should be deleted")

	recipients, err := file.Recipients(account)
	ASSERT_EQUAL(t, 0, len(recipients))
}

func TestListFiles(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	AddFile(account, strings.NewReader("hello world"), "hello.txt", []*Friend{})

	files := ListFiles(account, account.Fingerprint(), time.Now().Add(-time.Second), 10)
	ASSERT_EQUAL(t, 1, len(files))

	files = ListFiles(account, account.Fingerprint(), time.Now().Add(time.Second), 10)
	ASSERT_EQUAL(t, 0, len(files))

	files = ListFiles(account, "unknown fingerprint", time.Now().Add(-10*time.Second), 10)
	ASSERT_EQUAL(t, 0, len(files))
}
