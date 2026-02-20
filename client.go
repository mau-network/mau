package mau

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/go-resty/resty/v2"
)

var (
	ErrFriendNotFollowed        = errors.New("Friend is not being followed.")
	ErrCantFindFriend           = errors.New("Couldn't find friend.")
	ErrIncorrectPeerCertificate = errors.New("Incorrect Peer certificate.")
)

type Client struct {
	client  *resty.Client
	account *Account
	peer    Fingerprint
}

// TODO(maybe) Cache clients map[Fingerprint]*Client
func (a *Account) Client(peer Fingerprint, DNSNames []string) (*Client, error) {
	cert, err := a.certificate(DNSNames)
	if err != nil {
		return nil, err
	}

	c := &Client{
		account: a,
		peer:    peer,
	}

	c.client = resty.New().
		SetRedirectPolicy(resty.NoRedirectPolicy()).
		SetTimeout(httpClientTimeout).
		SetTLSClientConfig(
			&tls.Config{
				Certificates:          []tls.Certificate{cert},
				InsecureSkipVerify:    true,
				VerifyPeerCertificate: c.verifyPeerCertificate,
				// Go 1.26: Use secure defaults, removed explicit CipherSuites
				// Modern Go automatically selects optimal cipher suites
				MinVersion: tls.VersionTLS13, // TLS 1.3 for better security and performance
				CurvePreferences: []tls.CurveID{
					tls.X25519, // Modern, fast elliptic curve
					tls.CurveP256,
					tls.CurveP384,
				},
			},
		)

	return c, nil
}

func (c *Client) resolveFingerprintAddress(ctx context.Context, fingerprint Fingerprint, resolvers []FingerprintResolver) (string, error) {
	addresses := make(chan string, 1)

	// ask all resolvers for the address concurrently
	for _, fr := range resolvers {
		go func(resolver FingerprintResolver) {
			_ = resolver(ctx, fingerprint, addresses)
		}(fr)
	}

	select {
	case address := <-addresses:
		return address, nil
	case <-ctx.Done():
		return "", ErrCantFindFriend
	}
}

func (c *Client) fetchFileList(ctx context.Context, fingerprint Fingerprint, address string, after time.Time) ([]FileListItem, error) {
	var list []FileListItem

	resp, err := c.client.
		R().
		SetContext(ctx).
		SetHeader("If-Modified-Since", after.UTC().Format(http.TimeFormat)).
		SetResult(&list).
		ForceContentType("application/json").
		Get(
			(&url.URL{
				Scheme: uriProtocolName,
				Host:   address,
				Path:   fmt.Sprintf("/p2p/%s", fingerprint),
			}).String(),
		)

	if err != nil {
		return nil, fmt.Errorf("failed to request file list from peer %s: %w", fingerprint, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("peer %s responded with error status %s", fingerprint, resp.Status())
	}

	return list, nil
}

func (c *Client) downloadFiles(ctx context.Context, address string, fingerprint Fingerprint, list []FileListItem) error {
	for i := range list {
		select {
		case <-ctx.Done():
			return nil
		default:
			err := c.DownloadFile(ctx, address, fingerprint, &list[i])
			if err != nil {
				slog.Error("failed to download file", "file", list[i].Name, "error", err)
			}
		}
	}
	return nil
}

func (c *Client) DownloadFriend(ctx context.Context, fingerprint Fingerprint, after time.Time, fingerprintResolvers []FingerprintResolver) error {
	followed := path.Join(c.account.path, fingerprint.String())
	if _, err := os.Stat(followed); err != nil {
		return ErrFriendNotFollowed
	}

	address, err := c.resolveFingerprintAddress(ctx, fingerprint, fingerprintResolvers)
	if err != nil {
		return err
	}

	list, err := c.fetchFileList(ctx, fingerprint, address, after)
	if err != nil {
		return err
	}

	return c.downloadFiles(ctx, address, fingerprint, list)
}

func (c *Client) fileAlreadyExists(f *File, expectedSize int64, expectedHash string) bool {
	s, err := f.Size()
	if err != nil || s != expectedSize {
		return false
	}

	h, err := f.Hash()
	return err == nil && h == expectedHash
}

func (c *Client) downloadFileContent(ctx context.Context, address string, fingerprint Fingerprint, fileName string) ([]byte, error) {
	resp, err := c.client.
		R().
		SetContext(ctx).
		Get(
			(&url.URL{
				Scheme: uriProtocolName,
				Host:   address,
				Path:   fmt.Sprintf("/p2p/%s/%s", fingerprint, fileName),
			}).String(),
		)

	if err != nil {
		return nil, fmt.Errorf("failed to download file %s from peer %s: %w", fileName, fingerprint, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("server returned status %s while downloading file %s from %s", resp.Status(), fileName, resp.Request.URL)
	}

	return resp.Body(), nil
}

func validateDownloadedContent(data []byte, file *FileListItem) error {
	if len(data) != int(file.Size) {
		return fmt.Errorf("file size mismatch for %s: expected %d bytes, received %d bytes", file.Name, file.Size, len(data))
	}

	hash := sha256.Sum256(data)
	h := fmt.Sprintf("%x", hash)
	if h != file.Sum {
		return fmt.Errorf("file hash mismatch for %s: expected %s, received %s", file.Name, file.Sum, h)
	}

	return nil
}

func (c *Client) writeAndVerifyTemp(data []byte, tmpPath string, fingerprint Fingerprint) error {
	if err := os.WriteFile(tmpPath, data, FilePerm); err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	tmpFile := &File{Path: tmpPath}
	if err := tmpFile.VerifySignature(c.account, fingerprint); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

func createVersionBackup(f *File, tmpPath string) error {
	existingHash, err := f.Hash()
	if err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to hash existing file for versioning: %w", err)
	}

	versionsDir := f.Path + ".versions"
	if err := os.MkdirAll(versionsDir, DirPerm); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	versionPath := path.Join(versionsDir, existingHash)
	if err := os.Rename(f.Path, versionPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save file version: %w", err)
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, address string, fingerprint Fingerprint, file *FileListItem) error {
	f := File{
		Path:    path.Join(c.account.path, fingerprint.String(), file.Name),
		version: false,
	}

	// Check if file already exists with same content
	if c.fileAlreadyExists(&f, file.Size, file.Sum) {
		return nil
	}

	// Download file content
	data, err := c.downloadFileContent(ctx, address, fingerprint, file.Name)
	if err != nil {
		return err
	}

	// Validate downloaded content
	if err := validateDownloadedContent(data, file); err != nil {
		return err
	}

	// Write to temporary file and verify signature
	tmpPath := f.Path + ".tmp"
	if err := c.writeAndVerifyTemp(data, tmpPath, fingerprint); err != nil {
		return err
	}

	// Create version backup if file exists
	if _, err := os.Stat(f.Path); err == nil {
		if err := createVersionBackup(&f, tmpPath); err != nil {
			return err
		}
	}

	// Move verified temp file to final location
	if err := os.Rename(tmpPath, f.Path); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to save verified file: %w", err)
	}

	return nil
}

func (c *Client) verifyPeerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	for _, rawcert := range rawCerts {
		certs, err := x509.ParseCertificates(rawcert)
		if err != nil {
			return err
		}

		id, err := FingerprintFromCert(certs)
		if err != nil {
			return err
		}

		if id == c.peer {
			return nil
		}
	}

	// non of the certs include fingerprint
	return ErrIncorrectPeerCertificate
}
