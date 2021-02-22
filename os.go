package main

import "os"

func ensureDirectory(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		return err
	}

	os.MkdirAll(path, 0700)
	return nil
}
