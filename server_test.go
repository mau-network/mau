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
)

func TestServer(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "password")
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, account)

	friendAccount, err := NewAccount(t.TempDir(), "Friend of Ahmed", "friend@example.com", "password")
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, friendAccount)

	friendPub, err := friendAccount.Export()
	ASSERT_ERROR(t, nil, err)
	friend, err := account.AddFriend(bytes.NewBuffer(friendPub))
	ASSERT_ERROR(t, nil, err)

	server, err := NewServer(account)
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, server)

	listener, address := TempListener()

	go func() {
		err := server.Serve(*listener)
		ASSERT_ERROR(t, nil, err)
	}()

	list_account_files_url := fmt.Sprintf("%s/p2p/%s", address, account.Fingerprint())

	t.Run("GET "+list_account_files_url, func(t T) {

		t.Run("With no client cert", func(t T) {

			t.Run("With no files yet", func(t T) {
				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})
		})

		t.Run("With client cert not a friend", func(t T) {
			anotherAccount, _ := NewAccount(t.TempDir(), "Unknown", "unknown@example.com", "password")
			cert, err := anotherAccount.Certificate()
			ASSERT_ERROR(t, nil, err)

			oldTransport := http.DefaultClient.Transport
			defer func() { http.DefaultClient.Transport = oldTransport }()

			http.DefaultClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{*cert},
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
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})
		})

		t.Run("With client cert of a friend", func(t T) {
			cert, err := friendAccount.Certificate()
			ASSERT_ERROR(t, nil, err)

			oldTransport := http.DefaultClient.Transport
			defer func() { http.DefaultClient.Transport = oldTransport }()

			http.DefaultClient.Transport = &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{*cert},
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
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one private file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, "[]", string(body))
			})

			t.Run("With one shared file", func(t T) {
				file, err := account.AddFile(strings.NewReader("Hello world"), "hello.txt", []*Friend{friend})
				ASSERT_ERROR(t, nil, err)
				defer os.Remove(file.Path)

				req, _ := http.NewRequest("GET", list_account_files_url, nil)
				req.Header.Add("If-Modified-Since", time.Now().Add(-time.Second).UTC().Format(http.TimeFormat))

				resp, err := http.DefaultClient.Do(req)
				ASSERT_ERROR(t, nil, err)
				ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

				body, err := io.ReadAll(resp.Body)
				ASSERT_ERROR(t, nil, err)
				ASSERT(t, strings.Contains(string(body), "hello.txt.pgp"), "hello.txt.pgp not found in the response, Response: %s", body)
			})

		})
	})
}
