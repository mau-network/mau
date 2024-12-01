package mau

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"log"
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
		SetTimeout(3 * time.Second).
		SetTLSClientConfig(
			&tls.Config{
				Certificates:          []tls.Certificate{cert},
				InsecureSkipVerify:    true,
				VerifyPeerCertificate: c.verifyPeerCertificate,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
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
		go fr(ctx, fingerprint, addresses)
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
		return fmt.Errorf("Error requesting list of files: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Peer responded with error: %s", resp.Status())
	}

	// Download each file in order
	for i := range list {
		select {
		case <-ctx.Done():
			return nil
		default:
			err = c.DownloadFile(ctx, address, fingerprint, &list[i])
			if err != nil {
				log.Printf("Error: Downloading File %s\n\t%s", resp.Request.URL, err)
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
		return fmt.Errorf("Error getting file: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("Server returned unsuccessful response %s for %s", resp.Status(), resp.Request.URL)
	}

	if len(resp.Body()) != int(file.Size) {
		return fmt.Errorf("Size is different for %s\nexpected %d received %d\ncontent:\n%s", resp.Request.URL, file.Size, len(resp.Body()), resp.Body())
	}

	hash := sha256.Sum256(resp.Body())
	h := fmt.Sprintf("%x", hash)
	if h != file.Sum {
		return fmt.Errorf("Hash sum is different received %s", h)
	}

	// TODO: check for file signature
	// TODO: check for file encrypted to current user
	// TODO: keep existing version

	err = os.WriteFile(f.Path, resp.Body(), 0600)
	if err != nil {
		return err
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
