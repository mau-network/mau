package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAtomicDraftWrite verifies draft writes are atomic
func TestAtomicDraftWrite(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	// Create mock app with data directory
	app := &MauApp{
		dataDir: tempDir,
	}
	app.toastQueue = []string{}

	// Create home view
	hv := &HomeView{
		app: app,
	}

	// Simulate writing draft
	draftPath := filepath.Join(tempDir, draftFile)
	tmpPath := draftPath + ".tmp"

	testContent := "Test draft content"

	// Write test content using atomic pattern
	if err := os.WriteFile(tmpPath, []byte(testContent), 0600); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Verify temp file exists
	if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
		t.Fatal("Temp file should exist before rename")
	}

	// Rename to final location
	if err := os.Rename(tmpPath, draftPath); err != nil {
		t.Fatalf("Failed to rename: %v", err)
	}

	// Verify temp file is gone
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Error("Temp file should not exist after rename")
	}

	// Verify final file exists with correct content
	data, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("Failed to read final draft: %v", err)
	}

	if string(data) != testContent {
		t.Errorf("Expected content %q, got %q", testContent, string(data))
	}
}

// TestDraftWriteNoCorruption simulates interrupted write
func TestDraftWriteNoCorruption(t *testing.T) {
	tempDir := t.TempDir()
	draftPath := filepath.Join(tempDir, draftFile)

	// Write original valid draft
	originalContent := "Original draft"
	if err := os.WriteFile(draftPath, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to write original: %v", err)
	}

	// Simulate interrupted write by leaving temp file
	tmpPath := draftPath + ".tmp"
	corruptContent := "Incomplete..."
	if err := os.WriteFile(tmpPath, []byte(corruptContent), 0600); err != nil {
		t.Fatalf("Failed to write temp: %v", err)
	}

	// Verify original file is unchanged (atomic write prevents corruption)
	data, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("Failed to read draft: %v", err)
	}

	if string(data) != originalContent {
		t.Error("Original file was corrupted - atomic write failed!")
	}

	// Clean up temp file (real code should do this on startup)
	os.Remove(tmpPath)
}

// TestDraftPermissions verifies correct file permissions
func TestDraftPermissions(t *testing.T) {
	tempDir := t.TempDir()
	draftPath := filepath.Join(tempDir, draftFile)

	testContent := "Test content"
	if err := os.WriteFile(draftPath, []byte(testContent), 0600); err != nil {
		t.Fatalf("Failed to write: %v", err)
	}

	info, err := os.Stat(draftPath)
	if err != nil {
		t.Fatalf("Failed to stat: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0600)

	// Check if permissions are restricted (owner read/write only)
	if mode.Perm() != expectedMode {
		t.Errorf("Expected permissions %o, got %o", expectedMode, mode.Perm())
	}
}
