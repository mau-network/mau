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

func (c *Client) DownloadFriend(ctx context.Context, fingerprint Fingerprint, after time.Time, fingerprintResolvers []FingerprintResolver) error {
	followed := path.Join(c.account.path, fingerprint.String())
	if _, err := os.Stat(followed); err != nil {
		return ErrFriendNotFollowed
	}

	addresses := make(chan string, 1)

	// ask all resolvers for the address concurrently
	for _, fr := range fingerprintResolvers {
		go func(resolver FingerprintResolver) {
			_ = resolver(ctx, fingerprint, addresses)
		}(fr)
	}

	var address string
	select {
	case address = <-addresses:
	case <-ctx.Done():
		return ErrCantFindFriend
	}

	// Get list of remote files since the last modification we have
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
		return fmt.Errorf("failed to request file list from peer %s: %w", fingerprint, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("peer %s responded with error status %s", fingerprint, resp.Status())
	}

	// Download each file in order
	for i := range list {
		select {
		case <-ctx.Done():
			return nil
		default:
			err = c.DownloadFile(ctx, address, fingerprint, &list[i])
			if err != nil {
				slog.Error("failed to download file", "url", resp.Request.URL, "file", list[i].Name, "error", err)
			}
		}
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, address string, fingerprint Fingerprint, file *FileListItem) error {
	f := File{
		Path:    path.Join(c.account.path, fingerprint.String(), file.Name),
		version: false,
	}

	// check if it's the same file first by checking the size if it's not the same size then we can download it
	// if it's the same size then lets double check by summing it
	s, err := f.Size()
	if err == nil && s == file.Size {
		h, err := f.Hash()
		if err == nil && h == file.Sum {
			return nil
		}
	}

	resp, err := c.client.
		R().
		SetContext(ctx).
		Get(
			(&url.URL{
				Scheme: uriProtocolName,
				Host:   address,
				Path:   fmt.Sprintf("/p2p/%s/%s", fingerprint, file.Name),
			}).String(),
		)

	if err != nil {
		return fmt.Errorf("failed to download file %s from peer %s: %w", file.Name, fingerprint, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("server returned status %s while downloading file %s from %s", resp.Status(), file.Name, resp.Request.URL)
	}

	if len(resp.Body()) != int(file.Size) {
		return fmt.Errorf("file size mismatch for %s: expected %d bytes, received %d bytes", file.Name, file.Size, len(resp.Body()))
	}

	hash := sha256.Sum256(resp.Body())
	h := fmt.Sprintf("%x", hash)
	if h != file.Sum {
		return fmt.Errorf("file hash mismatch for %s: expected %s, received %s", file.Name, file.Sum, h)
	}

	// Write to temporary file first for verification
	tmpPath := f.Path + ".tmp"
	err = os.WriteFile(tmpPath, resp.Body(), FilePerm)
	if err != nil {
		return fmt.Errorf("failed to write temporary file: %w", err)
	}

	// Verify signature before accepting the file
	tmpFile := &File{Path: tmpPath}
	err = tmpFile.VerifySignature(c.account, fingerprint)
	if err != nil {
		os.Remove(tmpPath) // Clean up temporary file
		return fmt.Errorf("signature verification failed for %s: %w", file.Name, err)
	}

	// Signature verified, move temp file to final location
	// Before overwriting, save existing version if it exists
	if _, err := os.Stat(f.Path); err == nil {
		// File exists, create version backup
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
	}

	err = os.Rename(tmpPath, f.Path)
	if err != nil {
		os.Remove(tmpPath) // Clean up temporary file
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
