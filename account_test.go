package main

import (
	"path"
	"testing"
)

type T = *testing.T

func TestNewAccount(t *testing.T) {
	t.Run("Creating an account with valid parameters", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@gmail.com", "strong password")

		ASSERT(t, err == nil, "Error was returned when creating an account: %s", err)
		ASSERT(t, account != nil, "Account value is nil, expected a value")

		t.Run("Include correct information", func(t T) {
			ASSERT_EQUAL(t, "ahmed@gmail.com", account.Email())
			ASSERT_EQUAL(t, "Ahmed Mohamed", account.Name())
		})

		t.Run("Creates the correct file structure", func(t T) {
			ASSERT_DIR_EXISTS(t, path.Join(dir, ".mau"))
			ASSERT_FILE_EXISTS(t, path.Join(dir, ".mau", "account.pgp"))
		})
	})

	t.Run("Creating an account without a password", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ahmed Mohamed", "ahmed@gmail.com", "")

		ASSERT_ERROR(t, ErrPassphraseRequired, err)
		ASSERT_EQUAL(t, nil, account)
	})

	t.Run("Creating an account in an existing account directory", func(t T) {
		dir := t.TempDir()
		NewAccount(dir, "Ahmed Mohamed", "ahmed@gmail.com", "password")
		account, err := NewAccount(dir, "Ahmed Mahmoud", "ahmed.mahmoud@gmail.com", "password")

		ASSERT(t, err == ErrAccountAlreadyExists, "Expected an error: %s Got: %s", ErrAccountAlreadyExists, err)
		ASSERT(t, account == nil, "Expected the account to be nil value got : %v", account)
	})
}

func TestOpenAccount(t *testing.T) {
	dir := t.TempDir()
	account, _ := NewAccount(dir, "Ahmed Mohamed", "ahmed@gmail.com", "strong password")

	t.Run("Using same password", func(t T) {
		opened, err := OpenAccount(dir, "strong password")
		ASSERT_ERROR(t, nil, err)
		ASSERT_EQUAL(t, "ahmed@gmail.com", opened.Email())
		ASSERT_EQUAL(t, "Ahmed Mohamed", opened.Name())
		ASSERT_EQUAL(t, account.Fingerprint(), opened.Fingerprint())
	})
}
