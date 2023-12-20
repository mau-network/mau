package mau

import (
	"encoding/hex"
	"io"
	"path"
	"strings"
	"testing"
)

func TestFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")

	file, err := account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
	ASSERT_NO_ERROR(t, err)
	ASSERT_EQUAL(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp"), file.Path)

	versions := file.Versions()
	ASSERT_EQUAL(t, 0, len(versions))

	ASSERT_EQUAL(t, "hello.txt.pgp", file.Name())
	ASSERT_EQUAL(t, false, file.Deleted())

	size, err := file.Size()
	ASSERT_NO_ERROR(t, err)
	ASSERT(t, size > 0, "Size should not be zero")

	hash, err := file.Hash()
	ASSERT_NO_ERROR(t, err)
	REFUTE_EQUAL(t, "", hash)

	reader, err := file.Reader(account)
	ASSERT_NO_ERROR(t, err)

	content, err := io.ReadAll(reader)
	ASSERT_NO_ERROR(t, err)

	ASSERT_EQUAL(t, "hello world", string(content))

	friends, err := file.Recipients(account)
	ASSERT_NO_ERROR(t, err)
	ASSERT_EQUAL(t, 0, len(friends))

	t.Run("Versions", func(t T) {
		file, err := account.AddFile(strings.NewReader("hello there"), "hello.txt", []*Friend{})
		ASSERT_NO_ERROR(t, err)
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
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, 32, len(nameInbytes))

		gotVersion, err := account.GetFileVersion(account.Fingerprint(), "hello.txt.pgp", version.Name())
		ASSERT_NO_ERROR(t, err)
		ASSERT_EQUAL(t, version.Name(), gotVersion.Name())
		ASSERT_EQUAL(t, *version, *gotVersion)
	})
}
