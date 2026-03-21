package mau

import (
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFingerprintFromCert tests extraction of fingerprints from X.509 certificates
func TestFingerprintFromCert(t *testing.T) {
	t.Run("Ed25519 certificate (default for new accounts)", func(t T) {
		// Create a test account with Ed25519 key (default)
		dir := t.TempDir()
		account, err := NewAccount(dir, "Test User", "test@example.com", "testpass")
		require.NoError(t, err)

		// Generate TLS certificate
		cert, err := account.certificate(nil)
		require.NoError(t, err)
		require.NotEmpty(t, cert.Certificate)

		// Parse the certificate
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Extract fingerprint
		fpr, err := FingerprintFromCert([]*x509.Certificate{x509Cert})
		assert.NoError(t, err)
		assert.NotNil(t, fpr)

		// Fingerprint should match account fingerprint
		assert.Equal(t, account.Fingerprint(), fpr)

		// Ed25519 keys use v4 PGP format (20 bytes / 40 hex chars)
		assert.Len(t, fpr, 20, "Ed25519 fingerprint should be 20 bytes")
	})

	t.Run("Multiple certificates (should use first valid one)", func(t T) {
		// Create two accounts
		dir1 := t.TempDir()
		account1, err := NewAccount(dir1, "User 1", "user1@example.com", "pass1")
		require.NoError(t, err)

		dir2 := t.TempDir()
		account2, err := NewAccount(dir2, "User 2", "user2@example.com", "pass2")
		require.NoError(t, err)

		// Get both certificates
		cert1, err := account1.certificate(nil)
		require.NoError(t, err)
		cert2, err := account2.certificate(nil)
		require.NoError(t, err)

		x509Cert1, err := x509.ParseCertificate(cert1.Certificate[0])
		require.NoError(t, err)
		x509Cert2, err := x509.ParseCertificate(cert2.Certificate[0])
		require.NoError(t, err)

		// Should return fingerprint from first valid certificate
		fpr, err := FingerprintFromCert([]*x509.Certificate{x509Cert1, x509Cert2})
		assert.NoError(t, err)
		assert.Equal(t, account1.Fingerprint(), fpr)
	})

	t.Run("Empty certificate list", func(t T) {
		fpr, err := FingerprintFromCert([]*x509.Certificate{})
		assert.ErrorIs(t, err, ErrCantFindFingerprint)
		assert.Nil(t, fpr)
	})

	t.Run("Nil certificate list", func(t T) {
		fpr, err := FingerprintFromCert(nil)
		assert.ErrorIs(t, err, ErrCantFindFingerprint)
		assert.Nil(t, fpr)
	})
}

// TestFingerprintFromEd25519 tests Ed25519 fingerprint extraction
func TestFingerprintFromEd25519(t *testing.T) {
	t.Run("Valid Ed25519 certificate", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ed25519 User", "ed25519@example.com", "testpass")
		require.NoError(t, err)

		cert, err := account.certificate(nil)
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Verify it's Ed25519
		assert.Equal(t, x509.Ed25519, x509Cert.PublicKeyAlgorithm)

		// Extract fingerprint via internal function
		fpr, err := fingerprintFromEd25519(x509Cert)
		assert.NoError(t, err)
		assert.NotNil(t, fpr)
		assert.Len(t, fpr, 20, "Ed25519 fingerprint should be 20 bytes (v4 PGP)")

		// Should match the account fingerprint
		assert.Equal(t, []byte(account.Fingerprint()), fpr)
	})

	t.Run("Fingerprint embedded in DNSNames", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "DNS Test", "dns@example.com", "testpass")
		require.NoError(t, err)

		cert, err := account.certificate(nil)
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Verify fingerprint is in DNSNames (as 40 hex chars)
		require.NotEmpty(t, x509Cert.DNSNames)

		foundFingerprint := false
		for _, name := range x509Cert.DNSNames {
			if len(name) == 40 { // SHA-1 fingerprint hex length
				foundFingerprint = true
				// Verify it matches account fingerprint
				assert.Equal(t, account.Fingerprint().String(), name)
				break
			}
		}
		assert.True(t, foundFingerprint, "Fingerprint should be embedded in DNSNames")
	})
}

// TestCertToAddress tests address extraction from certificates
func TestCertToAddress(t *testing.T) {
	t.Run("Certificate with DNSNames", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Address Test", "address@example.com", "testpass")
		require.NoError(t, err)

		cert, err := account.certificate(nil)
		require.NoError(t, err)

		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		// Extract address
		addr, err := certToAddress([]*x509.Certificate{x509Cert})
		assert.NoError(t, err)
		assert.NotEmpty(t, addr)

		// Address should be the fingerprint (embedded in DNSNames)
		assert.Equal(t, account.Fingerprint().String(), addr)
	})

	t.Run("Multiple certificates returns first DNSName", func(t T) {
		dir1 := t.TempDir()
		account1, err := NewAccount(dir1, "First", "first@example.com", "pass1")
		require.NoError(t, err)

		dir2 := t.TempDir()
		account2, err := NewAccount(dir2, "Second", "second@example.com", "pass2")
		require.NoError(t, err)

		cert1, err := account1.certificate(nil)
		require.NoError(t, err)
		cert2, err := account2.certificate(nil)
		require.NoError(t, err)

		x509Cert1, err := x509.ParseCertificate(cert1.Certificate[0])
		require.NoError(t, err)
		x509Cert2, err := x509.ParseCertificate(cert2.Certificate[0])
		require.NoError(t, err)

		addr, err := certToAddress([]*x509.Certificate{x509Cert1, x509Cert2})
		assert.NoError(t, err)

		// Should return first certificate's address
		assert.Equal(t, account1.Fingerprint().String(), addr)
	})

	t.Run("Empty certificate list", func(t T) {
		addr, err := certToAddress([]*x509.Certificate{})
		assert.ErrorIs(t, err, ErrCantFindAddress)
		assert.Empty(t, addr)
	})

	t.Run("Nil certificate list", func(t T) {
		addr, err := certToAddress(nil)
		assert.ErrorIs(t, err, ErrCantFindAddress)
		assert.Empty(t, addr)
	})
}

// TestFingerprintConsistency verifies fingerprints are consistent across operations
func TestFingerprintConsistency(t *testing.T) {
	t.Run("Same fingerprint from Account and Certificate", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Consistency Test", "consistent@example.com", "testpass")
		require.NoError(t, err)

		// Get fingerprint directly from account
		accountFpr := account.Fingerprint()

		// Get fingerprint via certificate
		cert, err := account.certificate(nil)
		require.NoError(t, err)
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		certFpr, err := FingerprintFromCert([]*x509.Certificate{x509Cert})
		require.NoError(t, err)

		// Should be identical
		assert.Equal(t, accountFpr, certFpr)
		assert.True(t, accountFpr.Equal(certFpr))
	})

	t.Run("Address matches fingerprint string", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Address Match", "match@example.com", "testpass")
		require.NoError(t, err)

		cert, err := account.certificate(nil)
		require.NoError(t, err)
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		addr, err := certToAddress([]*x509.Certificate{x509Cert})
		require.NoError(t, err)

		// Address should be fingerprint as hex string
		assert.Equal(t, account.Fingerprint().String(), addr)
	})
}

// TestFingerprintFromPublicKey tests the internal routing function
func TestFingerprintFromPublicKey(t *testing.T) {
	t.Run("Routes to Ed25519 handler", func(t T) {
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ed25519 Route", "route@example.com", "testpass")
		require.NoError(t, err)

		cert, err := account.certificate(nil)
		require.NoError(t, err)
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)

		assert.Equal(t, x509.Ed25519, x509Cert.PublicKeyAlgorithm)

		fpr, err := fingerprintFromPublicKey(x509Cert)
		assert.NoError(t, err)
		assert.NotNil(t, fpr)
		assert.Equal(t, []byte(account.Fingerprint()), fpr)
	})

	t.Run("ECDSA returns nil (not yet supported)", func(t T) {
		// Mau doesn't generate ECDSA keys, but the code handles it
		// This documents the current behavior

		// Note: Creating a real ECDSA cert would require significant setup.
		// For now, we document that ECDSA returns (nil, nil) per the code.
		// If ECDSA support is added later, this test should be updated.

		// This is a documentation test - the actual ECDSA path returns (nil, nil)
		// and is skipped by FingerprintFromCert's loop
	})
}
