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
	"time"

	"github.com/stretchr/testify/assert"
)

func TestServer(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "password")
	assert.NoError(t, err)
	assert.NotEqual(t, nil, account)

	friendAccount, err := NewAccount(t.TempDir(), "Friend of Ahmed", "friend@example.com", "password")
	assert.NoError(t, err)
	assert.NotEqual(t, nil, friendAccount)

	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	assert.NoError(t, err)

	server, err := account.Server(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, nil, server)

	listener, address := TempListener()

	go func() {
		err := server.Serve(*listener, "")
		assert.Error(t, http.ErrServerClosed, err)
	}()
	defer server.Close()

	list_account_files_url := fmt.Sprintf("%s://%s/p2p/%s", uriProtocolName, address, account.Fingerprint())

	t.Run("GET "+list_account_files_url, func(t T) {

		t.Run("With no client cert", func(t T) {

			t.Run("With no files yet", func(t T) {
				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})
		})

		t.Run("With client cert not a friend", func(t T) {
			anotherAccount, _ := NewAccount(t.TempDir(), "Unknown", "unknown@example.com", "password")
			cert, err := anotherAccount.certificate(nil)
			assert.NoError(t, err)

			oldTransport := http.DefaultClient.Transport
			defer func() { http.DefaultClient.Transport = oldTransport }()

			http.DefaultClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
			}

			t.Run("With no files yet", func(t T) {
				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})
		})

		t.Run("With client cert of a friend", func(t T) {
			cert, err := friendAccount.certificate(nil)
			assert.NoError(t, err)

			oldTransport := http.DefaultClient.Transport
			defer func() { http.DefaultClient.Transport = oldTransport }()

			http.DefaultClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{cert},
					InsecureSkipVerify: true,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
			}

			t.Run("With no files yet", func(t T) {
				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Equal(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				assert.NoError(t, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				resp.Body.Close()
				assert.NoError(t, err)
				assert.Contains(t, string(body), "hello.txt.pgp", "hello.txt.pgp not found in the response, Response: %s", body)
			})

		})
	})
}

// TestServerRangeRequests verifies that HTTP Range requests work for resumable downloads
func TestServerRangeRequests(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Range Test User", "range@example.com", "password")
	assert.NoError(t, err)

	friendAccount, err := NewAccount(t.TempDir(), "Friend", "friend@example.com", "password")
	assert.NoError(t, err)

	var friendPub bytes.Buffer
	err = friendAccount.Export(&friendPub)
	assert.NoError(t, err)
	friend, err := account.AddFriend(&friendPub)
	assert.NoError(t, err)

	cert, err := friendAccount.certificate(nil)
	assert.NoError(t, err)

	server, err := account.Server(nil)
	assert.NoError(t, err)

	listener, address := TempListener()

	go func() {
		err := server.Serve(*listener, "")
		assert.Error(t, http.ErrServerClosed, err)
	}()
	defer server.Close()

	oldTransport := http.DefaultClient.Transport
	defer func() { http.DefaultClient.Transport = oldTransport }()

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates:       []tls.Certificate{cert},
			InsecureSkipVerify: true,
		},
	}

	testContent := "0123456789abcdefghijklmnopqrstuvwxyz"
	file, err := account.AddFile(strings.NewReader(testContent), "test-range.txt", []*Friend{friend})
	assert.NoError(t, err)
	defer os.Remove(file.Path)

	fileURL := fmt.Sprintf("%s://%s/p2p/%s/test-range.txt.pgp", uriProtocolName, address, account.Fingerprint())

	t.Run("Accept-Ranges header is set", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fileURL, nil)
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "bytes", resp.Header.Get("Accept-Ranges"))
	})

	t.Run("Partial content with Range header", func(t *testing.T) {
		req, _ := http.NewRequest("GET", fileURL, nil)
		req.Header.Set("Range", "bytes=0-9")

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusPartialContent, resp.StatusCode)
		assert.Contains(t, resp.Header.Get("Content-Range"), "bytes 0-9")
	})

	t.Run("Resume interrupted download", func(t *testing.T) {
		req1, _ := http.NewRequest("GET", fileURL, nil)
		req1.Header.Set("Range", "bytes=0-49")

		resp1, err := http.DefaultClient.Do(req1)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusPartialContent, resp1.StatusCode)
		body1, err := io.ReadAll(resp1.Body)
		resp1.Body.Close()
		assert.NoError(t, err)

		req2, _ := http.NewRequest("GET", fileURL, nil)
		req2.Header.Set("Range", "bytes=50-")

		resp2, err := http.DefaultClient.Do(req2)
		assert.NoError(t, err)
		assert.Equal(t, http.StatusPartialContent, resp2.StatusCode)

		body2, err := io.ReadAll(resp2.Body)
		resp2.Body.Close()
		assert.NoError(t, err)

		combined := append(body1, body2...)
		assert.NotEmpty(t, combined)
	})
}
