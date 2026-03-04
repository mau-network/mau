package mau

import (
	"encoding/hex"
	"errors"
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
		assert.NoError(t, err)
		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
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
		assert.NoError(t, err)
		content, err := io.ReadAll(reader)
		assert.NoError(t, err)
		assert.NoError(t, err)
		assert.Equal(t, "hello there", string(content))
	})
}

func TestContainsPathSeparator(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "simple filename",
			filename: "document.txt",
			expected: false,
		},
		{
			name:     "filename with spaces",
			filename: "my document.txt",
			expected: false,
		},
		{
			name:     "filename with forward slash",
			filename: "path/to/file.txt",
			expected: true,
		},
		{
			name:     "filename with backward slash",
			filename: "path\\to\\file.txt",
			expected: true,
		},
		{
			name:     "filename starting with slash",
			filename: "/etc/passwd",
			expected: true,
		},
		{
			name:     "filename with mixed slashes",
			filename: "path/to\\file.txt",
			expected: true,
		},
		{
			name:     "filename with special characters",
			filename: "file@#$%.txt",
			expected: false,
		},
		{
			name:     "unicode filename",
			filename: "文档.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsPathSeparator(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsRelativePathComponent(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{
			name:     "simple filename",
			filename: "document.txt",
			expected: false,
		},
		{
			name:     "current directory dot",
			filename: ".",
			expected: true,
		},
		{
			name:     "parent directory",
			filename: "..",
			expected: true,
		},
		{
			name:     "dot prefix with separator",
			filename: "./file.txt",
			expected: true,
		},
		{
			name:     "double dot prefix with separator",
			filename: "../file.txt",
			expected: true,
		},
		{
			name:     "hidden file",
			filename: ".hidden",
			expected: false,
		},
		{
			name:     "file starting with double dot",
			filename: "..config",
			expected: false,
		},
		{
			name:     "normal filename with dot",
			filename: "file.name.txt",
			expected: false,
		},
		{
			name:     "filename ending with dots",
			filename: "file..",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRelativePathComponent(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateFileName(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		expectError bool
		errorType   error
	}{
		{
			name:        "valid simple filename",
			filename:    "document.txt",
			expectError: false,
		},
		{
			name:        "valid filename with spaces",
			filename:    "my document.txt",
			expectError: false,
		},
		{
			name:        "valid filename with special chars",
			filename:    "file@#$%.txt",
			expectError: false,
		},
		{
			name:        "valid unicode filename",
			filename:    "文档.txt",
			expectError: false,
		},
		{
			name:        "valid hidden file",
			filename:    ".hidden",
			expectError: false,
		},
		{
			name:        "empty filename",
			filename:    "",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "filename with forward slash",
			filename:    "path/to/file.txt",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "filename with backward slash",
			filename:    "path\\to\\file.txt",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "absolute path",
			filename:    "/etc/passwd",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "current directory",
			filename:    ".",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "parent directory",
			filename:    "..",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "relative path with dot",
			filename:    "./file.txt",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "relative path with double dot",
			filename:    "../file.txt",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "path traversal attempt",
			filename:    "../../etc/passwd",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
		{
			name:        "windows path",
			filename:    "C:\\Windows\\System32",
			expectError: true,
			errorType:   ErrInvalidFileName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileName(tt.filename)
			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tt.errorType))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
