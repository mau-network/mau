package main

import (
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	dir := t.TempDir()
	account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@example.com", "password")
	ASSERT_ERROR(t, nil, err)
	REFUTE_EQUAL(t, nil, account)

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
		req, err := http.NewRequest("GET", list_account_files_url, nil)
		req.Header.Add("If-Modified-Since", time.Now().UTC().Format(http.TimeFormat))

		resp, err := http.DefaultClient.Do(req)
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, "[]", string(body))
	})
}
