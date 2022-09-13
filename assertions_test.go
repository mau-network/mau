package main

import (
	"errors"
	"os"
)

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
		ASSERT(t, false, `File "%s" should have existed but not found.`, file)
	} else if !stat.Mode().IsRegular() {
		ASSERT(t, false, `Expected "%s" to be Regular file instead was: %s`, file, stat.Mode().Type())
	}
}

func ASSERT_EQUAL[Param comparable](t T, expected, actual Param) {
	t.Helper()
	ASSERT(t, expected == actual, "Expected: %v, Actual: %v", expected, actual)
}

func ASSERT_ERROR(t T, expected, actual error) {
	t.Helper()
	ASSERT(t, errors.Is(actual, expected), "Error: %s is not a: %s", actual, expected)
}
