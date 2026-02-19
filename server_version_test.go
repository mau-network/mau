package mau

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServerVersionEndpoint(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Alice", "alice@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, account)

	friendAccount, err := NewAccount(t.TempDir(), "Bob", "bob@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, friendAccount)

	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	assert.NoError(t, err)

	server, err := account.Server(nil)
	assert.NoError(t, err)
	assert.NotNil(t, server)

	listener, address := TempListener()

	go func() {
		err := server.Serve(*listener, "")
		assert.Error(t, http.ErrServerClosed, err)
	}()
	defer server.Close()

	// Add a file with initial content
	fileContent := "Initial version"
	file, err := account.AddFile(strings.NewReader(fileContent), "test.txt", []*Friend{friend})
	assert.NoError(t, err)
	defer os.Remove(file.Path)

	// Update the file to create versions
	// The first version will be moved to .versions/hash1
	// The new content will be in the main file
	updatedContent := "Updated version"
	file, err = account.AddFile(strings.NewReader(updatedContent), "test.txt", []*Friend{friend})
	assert.NoError(t, err)

	// Get the hash of the CURRENT file (which is the updated version)
	currentHash, err := file.Hash()
	assert.NoError(t, err)

	// To get the first version's hash, we need to read it from the versions directory
	// The first version is in test.txt.pgp.versions/{hash}
	versionsDir := file.Path + ".versions"
	entries, err := os.ReadDir(versionsDir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1, "Should have exactly one old version")
	firstVersionHash := entries[0].Name()

	t.Run("Invalid path segments", func(t *testing.T) {
		tests := []struct {
			name           string
			path           string
			expectedStatus int
		}{
			// Path without hash matches getReg (not versionReg), returns 404
			{"Too few segments", fmt.Sprintf("/p2p/%s/test.txt.pgp.version", account.Fingerprint()), http.StatusNotFound},
			{"Too many segments", fmt.Sprintf("/p2p/%s/test.txt.pgp.version/%s/extra", account.Fingerprint(), firstVersionHash), http.StatusBadRequest},
			{"Empty path", "/p2p//test.txt.pgp.version/hash", http.StatusBadRequest},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				url := fmt.Sprintf("%s://%s%s", uriProtocolName, address, tt.path)
				req, err := http.NewRequest("GET", url, nil)
				assert.NoError(t, err)

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, resp.StatusCode)
				resp.Body.Close()
			})
		}
	})

	t.Run("Invalid fingerprint", func(t *testing.T) {
		url := fmt.Sprintf("%s://%s/p2p/invalid_fingerprint/test.txt.pgp.version/%s",
			uriProtocolName, address, firstVersionHash)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("File not found", func(t *testing.T) {
		t.Run("Wrong filename", func(t *testing.T) {
			url := fmt.Sprintf("%s://%s/p2p/%s/nonexistent.txt.pgp.version/%s",
				uriProtocolName, address, account.Fingerprint(), firstVersionHash)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("Wrong version hash", func(t *testing.T) {
			url := fmt.Sprintf("%s://%s/p2p/%s/test.txt.pgp.version/0000000000000000000000000000000000000000",
				uriProtocolName, address, account.Fingerprint())
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			resp.Body.Close()
		})
	})

	t.Run("Without client certificate", func(t *testing.T) {
		url := fmt.Sprintf("%s://%s/p2p/%s/test.txt.pgp.version/%s",
			uriProtocolName, address, account.Fingerprint(), firstVersionHash)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		// Without certificate, should be unauthorized
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("With friend certificate", func(t *testing.T) {
		// Create HTTP client with friend's certificate
		friendCert, err := friendAccount.certificate(nil)
		assert.NoError(t, err)

		oldTransport := http.DefaultClient.Transport
		defer func() { http.DefaultClient.Transport = oldTransport }()

		http.DefaultClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{friendCert},
				InsecureSkipVerify: true,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
			},
		}

		t.Run("Get first version (from .versions directory)", func(t *testing.T) {
			url := fmt.Sprintf("%s://%s/p2p/%s/test.txt.pgp.version/%s",
				uriProtocolName, address, account.Fingerprint(), firstVersionHash)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusOK, resp.StatusCode)

			// Verify headers
			assert.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
			assert.Equal(t, "bytes", resp.Header.Get("Accept-Ranges"))

			// Verify encrypted content is returned (not empty)
			// Files are encrypted, so we can't check plaintext content
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.NoError(t, err)
			assert.NotEmpty(t, body, "Should return encrypted file content")
		})

		t.Run("Cannot access current version via version endpoint", func(t *testing.T) {
			// The current/latest version is NOT in .versions/ directory
			// It's the main file, accessible via /p2p/{fpr}/{filename} (not .version)
			url := fmt.Sprintf("%s://%s/p2p/%s/test.txt.pgp.version/%s",
				uriProtocolName, address, account.Fingerprint(), currentHash)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			// Should be 404 because current version isn't in .versions/
			assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			resp.Body.Close()
		})

		t.Run("Range request support on old version", func(t *testing.T) {
			url := fmt.Sprintf("%s://%s/p2p/%s/test.txt.pgp.version/%s",
				uriProtocolName, address, account.Fingerprint(), firstVersionHash)

			// Request partial content (encrypted, so just verify range works)
			req, err := http.NewRequest("GET", url, nil)
			assert.NoError(t, err)
			req.Header.Add("Range", "bytes=0-50")

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.Equal(t, http.StatusPartialContent, resp.StatusCode)

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			assert.NoError(t, err)
			// Should return exactly 51 bytes (0-50 inclusive)
			assert.Len(t, body, 51)
		})
	})

	t.Run("Private file without permission", func(t *testing.T) {
		// Create a private file (no recipients) and update it to create a version
		privateFile, err := account.AddFile(strings.NewReader("private v1"), "private.txt", []*Friend{})
		assert.NoError(t, err)
		defer os.Remove(privateFile.Path)
		
		privateFile, err = account.AddFile(strings.NewReader("private v2"), "private.txt", []*Friend{})
		assert.NoError(t, err)

		// Get old version hash
		versionsDir := privateFile.Path + ".versions"
		entries, err := os.ReadDir(versionsDir)
		assert.NoError(t, err)
		assert.Len(t, entries, 1)
		privateVersion := entries[0].Name()

		// Try to access with friend's certificate (who is NOT a recipient)
		friendCert, err := friendAccount.certificate(nil)
		assert.NoError(t, err)

		oldTransport := http.DefaultClient.Transport
		defer func() { http.DefaultClient.Transport = oldTransport }()

		http.DefaultClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{friendCert},
				InsecureSkipVerify: true,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
			},
		}

		url := fmt.Sprintf("%s://%s/p2p/%s/private.txt.pgp.version/%s",
			uriProtocolName, address, account.Fingerprint(), privateVersion)
		req, err := http.NewRequest("GET", url, nil)
		assert.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
		resp.Body.Close()
	})
}

func TestServerVersionEndpointWithDeletedFile(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Charlie", "charlie@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, account)

	friendAccount, err := NewAccount(t.TempDir(), "Dave", "dave@example.com", "password")
	assert.NoError(t, err)
	assert.NotNil(t, friendAccount)

	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	assert.NoError(t, err)

	server, err := account.Server(nil)
	assert.NoError(t, err)
	assert.NotNil(t, server)

	listener, address := TempListener()

	go func() {
		err := server.Serve(*listener, "")
		assert.Error(t, http.ErrServerClosed, err)
	}()
	defer server.Close()

	// Add and update a file to create a version
	var file *File
	_, err = account.AddFile(strings.NewReader("version 1"), "deleted.txt", []*Friend{friend})
	assert.NoError(t, err)
	
	file, err = account.AddFile(strings.NewReader("version 2"), "deleted.txt", []*Friend{friend})
	assert.NoError(t, err)

	// Get the old version hash (from .versions directory)
	versionsDir := file.Path + ".versions"
	entries, err := os.ReadDir(versionsDir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	oldVersionHash := entries[0].Name()

	// Delete the file (this removes the current file and metadata)
	err = account.RemoveFile(file)
	assert.NoError(t, err)

	// Try to access the deleted version
	friendCert, err := friendAccount.certificate(nil)
	assert.NoError(t, err)

	oldTransport := http.DefaultClient.Transport
	defer func() { http.DefaultClient.Transport = oldTransport }()

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{friendCert},
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
	}

	url := fmt.Sprintf("%s://%s/p2p/%s/deleted.txt.pgp.version/%s",
		uriProtocolName, address, account.Fingerprint(), oldVersionHash)
	req, err := http.NewRequest("GET", url, nil)
	assert.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	assert.NoError(t, err)
	if err != nil {
		return // Make nilaway happy
	}
	// Should return 404 since file metadata is gone
	// Even though the .versions directory might still exist physically
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	resp.Body.Close()
}

