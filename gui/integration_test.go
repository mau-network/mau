package main

import (
	"fmt"
	"testing"
	"time"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// TestServerStartupLogic tests server startup
// Note: Skipped because serverRunning flag is set via glib.IdleAdd which requires GTK main loop
func TestServerStartupLogic(t *testing.T) {
	t.Skip("Requires GTK main loop for glib.IdleAdd - server creation tested elsewhere")

	tmpDir := t.TempDir()

	app := &MauApp{
		dataDir: tmpDir,
	}

	app.configMgr = NewConfigManager(tmpDir)
	app.accountMgr = NewAccountManager(tmpDir)

	if err := app.accountMgr.Init(); err != nil {
		t.Fatalf("Failed to init account: %v", err)
	}

	// Start server
	if err := app.startServer(); err != nil {
		t.Fatalf("Server failed to start: %v", err)
	}
	defer app.stopServer()

	// Server instance should be created immediately
	if app.server == nil {
		t.Error("Server instance should not be nil after startServer()")
	}
}

// TestConfigurationFlow tests config save/load cycle
func TestConfigurationFlow(t *testing.T) {
	tmpDir := t.TempDir()

	// Create and save config
	cfg1 := NewConfigManager(tmpDir)
	err := cfg1.Update(func(cfg *AppConfig) {
		cfg.DarkMode = true
		cfg.ServerPort = 9090
		cfg.AutoSync = true
		cfg.AutoSyncMinutes = 120
	})
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load in new instance
	cfg2 := NewConfigManager(tmpDir)
	loaded := cfg2.Get()

	// Verify all settings persisted
	tests := []struct {
		name     string
		got      interface{}
		expected interface{}
	}{
		{"DarkMode", loaded.DarkMode, true},
		{"ServerPort", loaded.ServerPort, 9090},
		{"AutoSync", loaded.AutoSync, true},
		{"AutoSyncMinutes", loaded.AutoSyncMinutes, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.got)
			}
		})
	}
}

// TestPostManagerIntegration tests post management
func TestPostManagerIntegration(t *testing.T) {
	tmpDir := t.TempDir()

	// Setup account
	accMgr := NewAccountManager(tmpDir)
	if err := accMgr.Init(); err != nil {
		t.Fatalf("Failed to init account: %v", err)
	}

	// Create post manager
	postMgr := NewPostManager(accMgr.Account())

	// Create a post
	author := Author{
		Type:        "Person",
		Name:        accMgr.Account().Name(),
		Email:       accMgr.Account().Email(),
		Fingerprint: accMgr.Account().Fingerprint().String(),
	}

	post := NewPost("Test post body", author, []string{"test"})

	// Save post
	if err := postMgr.Save(post); err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	// Verify cache stats
	size, capacity := postMgr.CacheStats()
	if capacity != cacheMaxSize {
		t.Errorf("Expected cache capacity %d, got %d", cacheMaxSize, capacity)
	}

	// Size should be at least 1 (the post we just saved)
	if size < 1 {
		t.Error("Cache should contain at least 1 post")
	}
}

// TestTextBufferOperations tests GTK text buffer
func TestTextBufferOperations(t *testing.T) {
	buffer := gtk.NewTextBuffer(nil)

	// Test set and get text
	testText := "Hello, World!"
	buffer.SetText(testText)

	start := buffer.StartIter()
	end := buffer.EndIter()
	got := buffer.Text(start, end, false)

	if got != testText {
		t.Errorf("Expected %q, got %q", testText, got)
	}

	// Test character count
	charCount := buffer.CharCount()
	if charCount != len(testText) {
		t.Errorf("Expected %d chars, got %d", len(testText), charCount)
	}

	// Test insert
	buffer.InsertAtCursor(" More")
	start = buffer.StartIter()
	end = buffer.EndIter()
	newText := buffer.Text(start, end, false)

	if newText != testText+" More" {
		t.Errorf("Expected %q, got %q", testText+" More", newText)
	}
}

// TestMarkdownRendering tests markdown conversion
func TestMarkdownRendering(t *testing.T) {
	renderer := NewMarkdownRenderer()

	tests := []struct {
		name  string
		input string
	}{
		{"Bold", "**bold text**"},
		{"Italic", "*italic text*"},
		{"Link", "[link](https://example.com)"},
		{"Code", "`code`"},
		{"Heading", "# Title"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			html := renderer.ToHTML(tt.input)
			if html == "" {
				t.Error("HTML output should not be empty")
			}

			pango := renderer.ToPango(tt.input)
			if pango == "" {
				t.Error("Pango output should not be empty")
			}
		})
	}
}

// TestCacheOperations tests post cache
func TestCacheOperations(t *testing.T) {
	cache := NewPostCache(5, 1*time.Hour)

	// Set and get
	post := Post{Body: "Test"}
	cache.Set("key1", post)

	retrieved, found := cache.Get("key1")
	if !found {
		t.Error("Post should be in cache")
	}
	if retrieved.Body != post.Body {
		t.Error("Body mismatch")
	}

	// Invalidate
	cache.Invalidate("key1")
	_, found = cache.Get("key1")
	if found {
		t.Error("Should be invalidated")
	}

	// Test LRU eviction
	for i := 0; i < 10; i++ {
		cache.Set(fmt.Sprintf("k%d", i), Post{Body: fmt.Sprintf("P%d", i)})
	}

	// First items should be evicted
	_, found = cache.Get("k0")
	if found {
		t.Error("k0 should be evicted")
	}

	// Clear
	cache.Clear()
	size, _ := cache.Stats()
	if size != 0 {
		t.Error("Cache should be empty after clear")
	}
}

// TestIdentityOperations tests identity management
func TestIdentityOperations(t *testing.T) {
	tmpDir := t.TempDir()

	accMgr := NewAccountManager(tmpDir)
	if err := accMgr.Init(); err != nil {
		t.Fatalf("Failed to init: %v", err)
	}

	acc := accMgr.Account()

	// List identities
	identities := acc.ListIdentities()
	if len(identities) == 0 {
		t.Fatal("Should have at least one identity")
	}

	// Get info
	if acc.Name() == "" {
		t.Error("Name should not be empty")
	}
	if acc.Email() == "" {
		t.Error("Email should not be empty")
	}

	// Check primary
	if !acc.IsPrimaryIdentity(identities[0]) {
		t.Error("First identity should be primary")
	}

	// Fingerprint
	fp := acc.Fingerprint().String()
	if len(fp) < 16 {
		t.Error("Fingerprint too short")
	}
}

// TestRetryBackoff tests retry mechanism
func TestRetryBackoff(t *testing.T) {
	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("temp error")
		}
		return nil
	}

	cfg := DefaultRetryConfig()
	cfg.InitialDelay = 5 * time.Millisecond
	cfg.MaxDelay = 50 * time.Millisecond

	start := time.Now()
	err := RetryOperation(cfg, operation)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("Should succeed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Should have some delay
	if elapsed < 10*time.Millisecond {
		t.Error("Retry delays too short")
	}
}

// TestPassphraseCache tests passphrase caching
func TestPassphraseCache(t *testing.T) {
	tmpDir := t.TempDir()

	accMgr := NewAccountManager(tmpDir)
	if err := accMgr.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Cache passphrase
	testPass := "test-passphrase-123"
	accMgr.CachePassphrase(testPass)

	// Retrieve (returns string, bool)
	cached, found := accMgr.GetCachedPassphrase()
	if !found {
		t.Error("Passphrase should be cached")
	}
	if cached != testPass {
		t.Errorf("Expected %q, got %q", testPass, cached)
	}
}
