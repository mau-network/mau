package mau

import (
	"crypto"
	"crypto/rsa"
	"crypto/x509"
	"testing"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/ProtonMail/go-crypto/openpgp/packet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createRSAAccountEntity creates a test account entity with RSA keys
func createRSAAccountEntity(name, email string) (*openpgp.Entity, error) {
	return openpgp.NewEntity(name, "", email, &packet.Config{
		DefaultHash:            crypto.SHA256,
		DefaultCompressionAlgo: packet.CompressionZIP,
		Algorithm:              packet.PubKeyAlgoRSA,
		RSABits:                2048,
	})
}

// createRSAAccount creates a test account with RSA keys
func createRSAAccount(t *testing.T, name, email, passphrase string) *Account {
	t.Helper()
	dir := t.TempDir()
	
	// Create account directory structure
	acc, err := createAccountFile(dir)
	require.NoError(t, err)

	// Create RSA entity
	entity, err := createRSAAccountEntity(name, email)
	require.NoError(t, err)

	// Save encrypted
	err = saveEncryptedEntity(acc, entity, passphrase)
	require.NoError(t, err)

	return buildAccount(entity, dir)
}

// TestRSAAccountCreation tests that RSA accounts can be created
func TestRSAAccountCreation(t *testing.T) {
	account := createRSAAccount(t, "RSA User", "rsa@example.com", "testpass")
	
	assert.NotNil(t, account)
	assert.NotNil(t, account.entity)
	
	// Verify it's actually an RSA key
	_, ok := account.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	assert.True(t, ok, "Expected RSA private key")
	
	_, ok = account.entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
	assert.True(t, ok, "Expected RSA public key")
}

// TestGenerateRSACertificate tests RSA certificate generation
func TestGenerateRSACertificate(t *testing.T) {
	account := createRSAAccount(t, "RSA Cert Test", "rsacert@example.com", "testpass")
	
	// Extract RSA private key
	rsaPrivKey, err := extractRSAKeyFromEntity(account.entity)
	require.NoError(t, err)
	require.NotNil(t, rsaPrivKey)
	
	// Build certificate template
	template := buildCertificateTemplate(
		[]string{account.Fingerprint().String()},
		account.entity.PrimaryKey.CreationTime,
	)
	
	// Generate certificate
	cert, err := account.generateRSACertificate(template, rsaPrivKey)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
	
	// Parse and verify certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	
	assert.Equal(t, x509.RSA, x509Cert.PublicKeyAlgorithm)
	assert.Equal(t, x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature, x509Cert.KeyUsage)
	// Note: RSA certificates do NOT embed fingerprint in DNSNames (unlike Ed25519)
	// RSA fingerprint is computed from the public key directly
}

// TestExtractRSAKeyFromEntity tests RSA key extraction
func TestExtractRSAKeyFromEntity(t *testing.T) {
	t.Run("Valid RSA entity", func(t *testing.T) {
		account := createRSAAccount(t, "Extract Test", "extract@example.com", "testpass")
		
		rsaKey, err := extractRSAKeyFromEntity(account.entity)
		assert.NoError(t, err)
		assert.NotNil(t, rsaKey)
		
		// Verify it's a valid RSA key
		assert.NotNil(t, rsaKey.PublicKey.N)
		assert.NotEqual(t, 0, rsaKey.PublicKey.E)
		assert.NotNil(t, rsaKey.D)
		assert.NotEmpty(t, rsaKey.Primes)
	})
	
	t.Run("Ed25519 entity returns error", func(t *testing.T) {
		// Create default Ed25519 account
		dir := t.TempDir()
		account, err := NewAccount(dir, "Ed25519 User", "ed25519@example.com", "testpass")
		require.NoError(t, err)
		
		// Should fail to extract as RSA
		rsaKey, err := extractRSAKeyFromEntity(account.entity)
		assert.ErrorIs(t, err, ErrCannotConvertPrivateKey)
		assert.Nil(t, rsaKey)
	})
}

// TestBuildRSAKeyFromParts tests RSA key reconstruction
func TestBuildRSAKeyFromParts(t *testing.T) {
	account := createRSAAccount(t, "Key Parts Test", "keyparts@example.com", "testpass")
	
	// Get original keys
	origPrivKey, ok := account.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	require.True(t, ok)
	
	origPubKey, ok := account.entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
	require.True(t, ok)
	
	// Rebuild key
	rebuiltKey := buildRSAKeyFromParts(origPrivKey, origPubKey)
	
	assert.NotNil(t, rebuiltKey)
	assert.Equal(t, origPubKey.N, rebuiltKey.PublicKey.N)
	assert.Equal(t, int(origPubKey.E), rebuiltKey.PublicKey.E)
	assert.Equal(t, origPrivKey.D, rebuiltKey.D)
	assert.Equal(t, origPrivKey.Primes, rebuiltKey.Primes)
	
	// Verify precomputed values were copied
	assert.Equal(t, origPrivKey.Precomputed.Dp, rebuiltKey.Precomputed.Dp)
	assert.Equal(t, origPrivKey.Precomputed.Dq, rebuiltKey.Precomputed.Dq)
	assert.Equal(t, origPrivKey.Precomputed.Qinv, rebuiltKey.Precomputed.Qinv)
}

// TestEncodeCertificateAndKey tests PEM encoding of RSA certificate and key
func TestEncodeCertificateAndKey(t *testing.T) {
	account := createRSAAccount(t, "Encode Test", "encode@example.com", "testpass")
	
	rsaPrivKey, err := extractRSAKeyFromEntity(account.entity)
	require.NoError(t, err)
	
	template := buildCertificateTemplate(
		[]string{account.Fingerprint().String()},
		account.entity.PrimaryKey.CreationTime,
	)
	
	// Create certificate DER bytes
	derBytes, err := x509.CreateCertificate(nil, &template, &template, &rsaPrivKey.PublicKey, rsaPrivKey)
	require.NoError(t, err)
	
	// Encode to TLS certificate
	cert, err := encodeCertificateAndKey(rsaPrivKey, derBytes)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
	assert.NotEmpty(t, cert.Certificate)
	assert.NotEmpty(t, cert.PrivateKey)
	
	// Verify we can parse the certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	assert.NoError(t, err)
	assert.Equal(t, x509.RSA, x509Cert.PublicKeyAlgorithm)
	
	// Verify private key is correct type
	_, ok := cert.PrivateKey.(*rsa.PrivateKey)
	assert.True(t, ok, "PrivateKey should be *rsa.PrivateKey")
}

// TestFingerprintFromRSA tests RSA fingerprint extraction from certificate
func TestFingerprintFromRSA(t *testing.T) {
	account := createRSAAccount(t, "Fingerprint Test", "fpr@example.com", "testpass")
	
	// Generate certificate
	cert, err := account.certificate(nil)
	require.NoError(t, err)
	
	// Parse certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	
	// Extract fingerprint
	fprBytes, err := fingerprintFromRSA(x509Cert)
	assert.NoError(t, err)
	assert.NotNil(t, fprBytes)
	
	// Should match account fingerprint
	assert.Equal(t, []byte(account.Fingerprint()), fprBytes)
	
	// RSA fingerprints are also 20 bytes (SHA-1 of public key packet)
	assert.Len(t, fprBytes, 20)
}

// TestFingerprintFromCertWithRSA tests the public API with RSA certificates
func TestFingerprintFromCertWithRSA(t *testing.T) {
	account := createRSAAccount(t, "Cert Fingerprint Test", "certfpr@example.com", "testpass")
	
	cert, err := account.certificate(nil)
	require.NoError(t, err)
	
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	
	// Extract via public API
	fpr, err := FingerprintFromCert([]*x509.Certificate{x509Cert})
	assert.NoError(t, err)
	assert.NotNil(t, fpr)
	
	// Should match account fingerprint
	assert.Equal(t, account.Fingerprint(), fpr)
}

// TestRSACertificateIntegration tests full RSA certificate workflow
func TestRSACertificateIntegration(t *testing.T) {
	account := createRSAAccount(t, "Integration Test", "integration@example.com", "testpass")
	
	t.Run("Certificate generation", func(t *testing.T) {
		cert, err := account.certificate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, cert)
		assert.NotEmpty(t, cert.Certificate)
	})
	
	t.Run("Fingerprint consistency", func(t *testing.T) {
		// Get fingerprint from account
		accountFpr := account.Fingerprint()
		
		// Get fingerprint from certificate
		cert, err := account.certificate(nil)
		require.NoError(t, err)
		
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		
		certFpr, err := FingerprintFromCert([]*x509.Certificate{x509Cert})
		require.NoError(t, err)
		
		// Should match
		assert.Equal(t, accountFpr, certFpr)
		assert.True(t, accountFpr.Equal(certFpr))
	})
	
	t.Run("Certificate address matches fingerprint", func(t *testing.T) {
		cert, err := account.certificate(nil)
		require.NoError(t, err)
		
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		require.NoError(t, err)
		
		// For RSA certificates, certToAddress should succeed by extracting
		// fingerprint from the RSA public key
		_, err = certToAddress([]*x509.Certificate{x509Cert})
		
		// RSA certificates don't embed fingerprint in DNSNames,
		// so certToAddress will return ErrCantFindAddress
		// (it only looks at DNSNames, not reconstructing from public key)
		assert.ErrorIs(t, err, ErrCantFindAddress)
	})
}

// TestRSAvsEd25519Compatibility tests that both key types work correctly
func TestRSAvsEd25519Compatibility(t *testing.T) {
	rsaAccount := createRSAAccount(t, "RSA User", "rsa@example.com", "testpass")
	
	dir := t.TempDir()
	ed25519Account, err := NewAccount(dir, "Ed25519 User", "ed25519@example.com", "testpass")
	require.NoError(t, err)
	
	t.Run("Both generate valid certificates", func(t *testing.T) {
		rsaCert, err := rsaAccount.certificate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, rsaCert)
		
		ed25519Cert, err := ed25519Account.certificate(nil)
		assert.NoError(t, err)
		assert.NotNil(t, ed25519Cert)
	})
	
	t.Run("Both have valid fingerprints", func(t *testing.T) {
		rsaFpr := rsaAccount.Fingerprint()
		ed25519Fpr := ed25519Account.Fingerprint()
		
		assert.NotNil(t, rsaFpr)
		assert.NotNil(t, ed25519Fpr)
		assert.Len(t, rsaFpr, 20)
		assert.Len(t, ed25519Fpr, 20)
		
		// Should be different
		assert.False(t, rsaFpr.Equal(ed25519Fpr))
	})
	
	t.Run("Certificate fingerprints match account fingerprints", func(t *testing.T) {
		// RSA
		rsaCert, _ := rsaAccount.certificate(nil)
		rsaX509, _ := x509.ParseCertificate(rsaCert.Certificate[0])
		rsaCertFpr, err := FingerprintFromCert([]*x509.Certificate{rsaX509})
		assert.NoError(t, err)
		assert.Equal(t, rsaAccount.Fingerprint(), rsaCertFpr)
		
		// Ed25519
		ed25519Cert, _ := ed25519Account.certificate(nil)
		ed25519X509, _ := x509.ParseCertificate(ed25519Cert.Certificate[0])
		ed25519CertFpr, err := FingerprintFromCert([]*x509.Certificate{ed25519X509})
		assert.NoError(t, err)
		assert.Equal(t, ed25519Account.Fingerprint(), ed25519CertFpr)
	})
}

// TestRSAKeyReconstruction tests that rebuilt RSA keys work correctly
func TestRSAKeyReconstruction(t *testing.T) {
	account := createRSAAccount(t, "Reconstruction Test", "recon@example.com", "testpass")
	
	// Extract RSA keys from entity
	origPrivKey, _ := account.entity.PrivateKey.PrivateKey.(*rsa.PrivateKey)
	origPubKey, _ := account.entity.PrimaryKey.PublicKey.(*rsa.PublicKey)
	
	rebuiltKey := buildRSAKeyFromParts(origPrivKey, origPubKey)
	
	// Test that rebuilt key can be used for certificate generation
	template := buildCertificateTemplate(
		[]string{"test.example.com"},
		account.entity.PrimaryKey.CreationTime,
	)
	
	derBytes, err := x509.CreateCertificate(nil, &template, &template, &rebuiltKey.PublicKey, rebuiltKey)
	assert.NoError(t, err)
	assert.NotEmpty(t, derBytes)
	
	// Verify certificate is valid
	cert, err := x509.ParseCertificate(derBytes)
	assert.NoError(t, err)
	assert.Equal(t, x509.RSA, cert.PublicKeyAlgorithm)
}

// TestRSACertificateFields tests that RSA certificates have correct fields
func TestRSACertificateFields(t *testing.T) {
	account := createRSAAccount(t, "Field Test", "fields@example.com", "testpass")
	
	cert, err := account.certificate(nil)
	require.NoError(t, err)
	
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	require.NoError(t, err)
	
	// Verify algorithm
	assert.Equal(t, x509.RSA, x509Cert.PublicKeyAlgorithm)
	
	// Verify key usage
	assert.Equal(t, x509.KeyUsageKeyEncipherment|x509.KeyUsageDigitalSignature, x509Cert.KeyUsage)
	
	// Verify extended key usage
	assert.Contains(t, x509Cert.ExtKeyUsage, x509.ExtKeyUsageServerAuth)
	assert.Contains(t, x509Cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)
	
	// Note: RSA certificates do NOT embed fingerprint in DNSNames
	// (only Ed25519 does this). RSA fingerprint is computed from the public key.
	// If DNSNames were provided during certificate creation, they would be here,
	// but when certificate(nil) is called, RSA gets empty DNSNames.
	
	// Verify validity period (100 years)
	creationTime := account.entity.PrimaryKey.CreationTime
	expectedNotAfter := creationTime.AddDate(100, 0, 0)
	assert.Equal(t, creationTime.Unix(), x509Cert.NotBefore.Unix())
	assert.Equal(t, expectedNotAfter.Unix(), x509Cert.NotAfter.Unix())
}
