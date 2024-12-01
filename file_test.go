package mau

import (
	"encoding/hex"
	"io"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {
	account_dir := t.TempDir()
	account, _ := NewAccount(account_dir, "Ahmed Mohamed", "ahmed@example.com", "password value")
	var file *File
	var err error

	t.Run("New file", func(t T) {
		file, err = account.AddFile(strings.NewReader("hello world"), "hello.txt", []*Friend{})
		assert.NoError(t, err)
		assert.Equal(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp"), file.Path)

		t.Run("No versions", func(t T) {
			versions := file.Versions()
			assert.Equal(t, 0, len(versions))
		})

		t.Run("Not deleted", func(t T) {
			assert.Equal(t, "hello.txt.pgp", file.Name())
			assert.Equal(t, false, file.Deleted())
		})

		t.Run("Has data", func(t T) {
			size, err := file.Size()
			assert.NoError(t, err)
			assert.Greater(t, size, int64(0), "Size should not be zero")
		})

		t.Run("Has a hash", func(t T) {
			hash, err := file.Hash()
			assert.NoError(t, err)
			assert.NotEqual(t, "", hash)
		})

		t.Run("Can read same data back", func(t T) {
			reader, err := file.Reader(account)
			assert.NoError(t, err)

			content, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.Equal(t, "hello world", string(content))
		})

		t.Run("Has no recepients", func(t T) {
			friends, err := file.Recipients(account)
			assert.NoError(t, err)
			assert.Equal(t, 0, len(friends))
		})
	})

	t.Run("Versions", func(t T) {
		file, err := account.AddFile(strings.NewReader("hello there"), "hello.txt", []*Friend{})
		assert.NoError(t, err)
		assert.Equal(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp"), file.Path)

		versions := file.Versions()
		assert.Equal(t, 1, len(versions))
		assert.DirExists(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp.versions"))

		version := versions[0]
		assert.FileExists(t, path.Join(account_dir, account.Fingerprint().String(), "hello.txt.pgp.versions", version.Name()))

		reader, err := version.Reader(account)
		content, err := io.ReadAll(reader)
		assert.Equal(t, "hello world", string(content))

		nameInbytes, err := hex.DecodeString(version.Name())
		assert.NoError(t, err)
		assert.Equal(t, 32, len(nameInbytes))

		gotVersion, err := account.GetFileVersion(account.Fingerprint(), "hello.txt.pgp", version.Name())
		assert.NoError(t, err)
		assert.Equal(t, version.Name(), gotVersion.Name())
		assert.Equal(t, *version, *gotVersion)
	})

	t.Run("GetFile", func(t T) {
		file, err := account.GetFile(account.Fingerprint(), "hello.txt.pgp")
		assert.NoError(t, err)

		reader, err := file.Reader(account)
		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.Equal(t, "hello there", string(content))
	})
}
