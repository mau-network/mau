package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/mau-network/mau"
)

// TestCLIHelp verifies that running mau without arguments shows the help message
func TestCLIHelp(t *testing.T) {
	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Save original Stdout and restore after test
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	// Capture exit to prevent actual exit
	oldExit := exitFunc
	exitCode := 0
	exitFunc = func(code int) {
		exitCode = code
	}
	defer func() { exitFunc = oldExit }()

	// Run with no arguments
	os.Args = []string{"mau"}

	main()

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify exit code
	if exitCode != 1 {
		t.Errorf("Expected exit code 1, got %d", exitCode)
	}

	// Verify help message contains commands
	expectedCommands := []string{
		"init",
		"show",
		"export",
		"friend",
		"friends",
		"unfriend",
		"follow",
		"unfollow",
		"follows",
		"share",
		"files",
		"open",
		"delete",
		"serve",
		"sync",
	}

	for _, cmd := range expectedCommands {
		if !strings.Contains(output, cmd) {
			t.Errorf("Help output missing command: %s", cmd)
		}
	}
}

// TestCLIInit tests the init command
func TestCLIInit(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "mau-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args and restore after test
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	// Run init command
	os.Args = []string{"mau", "init", "-name", "Test User", "-email", "test@example.com"}

	main()

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	if !strings.Contains(output, "Initializing account") {
		t.Errorf("Expected 'Initializing account' in output")
	}
	if !strings.Contains(output, "Done") {
		t.Errorf("Expected 'Done' in output")
	}

	// Verify account directory was created
	accountDir := filepath.Join(tmpDir, ".mau")
	if _, err := os.Stat(accountDir); os.IsNotExist(err) {
		t.Errorf("Account directory was not created: %s", accountDir)
	}

	// Verify private key exists
	privateKeyPath := filepath.Join(accountDir, "account.pgp")
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Errorf("Private key was not created: %s", privateKeyPath)
	}
}

// TestCLIShow tests the show command
func TestCLIShow(t *testing.T) {
	// Create temporary directory with an account
	tmpDir, account := createTestAccount(t)
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	// Run show command
	os.Args = []string{"mau", "show"}

	main()

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output contains account info
	if !strings.Contains(output, "Name:") {
		t.Errorf("Expected 'Name:' in output")
	}
	if !strings.Contains(output, "Test User") {
		t.Errorf("Expected 'Test User' in output")
	}
	if !strings.Contains(output, "Email:") {
		t.Errorf("Expected 'Email:' in output")
	}
	if !strings.Contains(output, "test@example.com") {
		t.Errorf("Expected 'test@example.com' in output")
	}
	if !strings.Contains(output, "Fingerprint:") {
		t.Errorf("Expected 'Fingerprint:' in output")
	}
	if !strings.Contains(output, account.Fingerprint().String()) {
		t.Errorf("Expected fingerprint in output")
	}
}

// TestCLIExport tests the export command
func TestCLIExport(t *testing.T) {
	// Create temporary directory with an account
	tmpDir, _ := createTestAccount(t)
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Create output file path
	outputFile := filepath.Join(tmpDir, "public-key.asc")

	// Run export command
	os.Args = []string{"mau", "export", "-output", outputFile}

	main()

	// Verify output file was created
	if _, err := os.Stat(outputFile); os.IsNotExist(err) {
		t.Fatalf("Export file was not created: %s", outputFile)
	}

	// Read and verify content
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read export file: %v", err)
	}

	publicKeyContent := string(content)
	if !strings.Contains(publicKeyContent, "BEGIN PGP PUBLIC KEY BLOCK") {
		t.Errorf("Export file does not contain PGP public key block")
	}
}

// TestCLIFriendsWorkflow tests adding, listing, and removing friends
func TestCLIFriendsWorkflow(t *testing.T) {
	// Create two accounts for testing
	tmpDir1, _ := createTestAccount(t)
	defer os.RemoveAll(tmpDir1)

	tmpDir2, account2 := createTestAccount(t)
	defer os.RemoveAll(tmpDir2)

	// Export account2's public key
	keyFile := filepath.Join(tmpDir1, "friend-key.asc")
	f, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := account2.Export(f); err != nil {
		t.Fatalf("Failed to export key: %v", err)
	}
	f.Close()

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to account1's directory
	if err := os.Chdir(tmpDir1); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test: Add friend
	os.Args = []string{"mau", "friend", "-key", keyFile}
	main()

	// Close writer and read output
	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Friend added:") {
		t.Errorf("Expected 'Friend added:' in output, got: %s", output)
	}

	// Restore stdout for next command
	r, w, _ = os.Pipe()
	os.Stdout = w

	// Test: List friends
	os.Args = []string{"mau", "friends"}
	main()

	w.Close()
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	if !strings.Contains(output, "Test User") {
		t.Errorf("Expected friend name in friends list")
	}

	// Restore stdout for cleanup
	os.Stdout = oldStdout
}

// TestCLIFollowWorkflow tests following and unfollowing friends
func TestCLIFollowWorkflow(t *testing.T) {
	// Create two accounts for testing
	tmpDir1, _ := createTestAccount(t)
	defer os.RemoveAll(tmpDir1)

	tmpDir2, account2 := createTestAccount(t)
	defer os.RemoveAll(tmpDir2)

	// Export account2's public key and add as friend
	keyFile := filepath.Join(tmpDir1, "friend-key.asc")
	f, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := account2.Export(f); err != nil {
		t.Fatalf("Failed to export key: %v", err)
	}
	f.Close()

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to account1's directory
	if err := os.Chdir(tmpDir1); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Add friend first
	os.Args = []string{"mau", "friend", "-key", keyFile}
	main()

	// Capture stdout for follow command
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test: Follow friend
	fpr := account2.Fingerprint().String()
	os.Args = []string{"mau", "follow", "-fingerprint", fpr}
	main()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Following friend:") {
		t.Errorf("Expected 'Following friend:' in output, got: %s", output)
	}

	// Restore stdout for follows list
	r, w, _ = os.Pipe()
	os.Stdout = w

	// Test: List follows
	os.Args = []string{"mau", "follows"}
	main()

	w.Close()
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	if !strings.Contains(output, "Test User") {
		t.Errorf("Expected followed friend in follows list")
	}

	// Restore stdout for unfollow
	r, w, _ = os.Pipe()
	os.Stdout = w

	// Test: Unfollow friend
	os.Args = []string{"mau", "unfollow", "-fingerprint", fpr}
	main()

	w.Close()
	buf.Reset()
	io.Copy(&buf, r)
	output = buf.String()

	if !strings.Contains(output, "Unfollowing friend:") {
		t.Errorf("Expected 'Unfollowing friend:' in output")
	}

	// Restore stdout
	os.Stdout = oldStdout
}

// TestCLIUnfriend tests removing a friend
func TestCLIUnfriend(t *testing.T) {
	// Create two accounts
	tmpDir1, _ := createTestAccount(t)
	defer os.RemoveAll(tmpDir1)

	tmpDir2, account2 := createTestAccount(t)
	defer os.RemoveAll(tmpDir2)

	// Export account2's public key and add as friend
	keyFile := filepath.Join(tmpDir1, "friend-key.asc")
	f, err := os.Create(keyFile)
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	if err := account2.Export(f); err != nil {
		t.Fatalf("Failed to export key: %v", err)
	}
	f.Close()

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to account1's directory
	if err := os.Chdir(tmpDir1); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Add friend first
	os.Args = []string{"mau", "friend", "-key", keyFile}
	main()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test: Unfriend
	fpr := account2.Fingerprint().String()
	os.Args = []string{"mau", "unfriend", "-fingerprint", fpr}
	main()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Removing friend:") {
		t.Errorf("Expected 'Removing friend:' in output")
	}
	if !strings.Contains(output, "Done") {
		t.Errorf("Expected 'Done' in output")
	}

	// Restore stdout
	os.Stdout = oldStdout
}

// TestCLIFilesEmpty tests listing files when no files exist
func TestCLIFilesEmpty(t *testing.T) {
	// Create test account
	tmpDir, _ := createTestAccount(t)
	defer os.RemoveAll(tmpDir)

	// Save original working directory
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)

	// Change to temp directory
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}

	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Mock password input
	oldPasswordFunc := getPasswordFunc
	getPasswordFunc = func() string {
		return "test-passphrase"
	}
	defer func() { getPasswordFunc = oldPasswordFunc }()

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Test: List files (should be empty)
	os.Args = []string{"mau", "files"}
	main()

	w.Close()
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Should complete without error (empty output is expected)
	_ = output

	// Restore stdout
	os.Stdout = oldStdout
}

// Helper function to create a test account
func createTestAccount(t *testing.T) (string, *Account) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "mau-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	account, err := NewAccount(tmpDir, "Test User", "test@example.com", "test-passphrase")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create test account: %v", err)
	}

	return tmpDir, account
}
