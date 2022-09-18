package mau

import (
	"crypto/tls"
	"errors"
	"net/http"
	"os"
	"path"
	"testing"
)

type T = *testing.T

func init() {
	rsaKeyLength = 1024 // for faster account generation

	http.DefaultClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
			CipherSuites: []uint16{
				tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			},
		},
	}
}

func ASSERT(t T, condition bool, message string, args ...interface{}) {
	t.Helper()
	if !condition {
		t.Errorf(message, args...)
	}
}

func ASSERT_DIR_EXISTS(t T, dir string) {
	t.Helper()
	if stat, err := os.Stat(dir); os.IsNotExist(err) {
		ASSERT(t, false, `Directory "%s" should have existed but not found.`, dir)
	} else if !stat.IsDir() {
		ASSERT(t, false, `Expected "%s" to be directory instead was: %s`, dir, stat.Mode().Type())
	}
}

func ASSERT_FILE_EXISTS(t T, file string) {
	t.Helper()
	if stat, err := os.Stat(file); os.IsNotExist(err) {

		parent := path.Dir(file)
		siblings := []string{}
		entries, err := os.ReadDir(parent)
		ASSERT_NO_ERROR(t, err)
		for _, entry := range entries {
			siblings = append(siblings, entry.Name())
		}

		ASSERT(t, false, "File \"%s\" should have existed but not found.\nOther files in same directory: %s", file, siblings)
	} else if !stat.Mode().IsRegular() {
		ASSERT(t, false, `Expected "%s" to be Regular file instead was: %s`, file, stat.Mode().Type())
	}
}

func REFUTE_FILE_EXISTS(t T, file string) {
	t.Helper()
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return
	}

	ASSERT(t, false, `File "%s" shouldn't have existed but it was found.`, file)
}

func ASSERT_EQUAL[Param comparable](t T, expected, actual Param) {
	t.Helper()
	ASSERT(t, expected == actual, "Expected: %v, Actual: %v", expected, actual)
}

func ASSERT_ERROR(t T, expected, actual error) {
	t.Helper()
	ASSERT(t, errors.Is(actual, expected), "Error: %s is not a: %s", actual, expected)
}

func ASSERT_NO_ERROR(t T, actual error) {
	t.Helper()
	ASSERT(t, actual == nil, "Expected no error found: %s", actual)
}

func REFUTE_EQUAL[Param comparable](t T, expected, actual Param) {
	t.Helper()
	ASSERT(t, expected != actual, "Expected values not to be equal, Expected: %v Actual: %v", expected, actual)
}
