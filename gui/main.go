package main

import (
	"os"
	"path/filepath"
)

func main() {
	// Allow data dir to be overridden via environment variable for testing
	dataDir := os.Getenv("MAU_GUI_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(os.Getenv("HOME"), ".mau-gui")
	}

	app := NewMauApp(dataDir)

	if code := app.Run(os.Args); code > 0 {
		os.Exit(code)
	}
}
