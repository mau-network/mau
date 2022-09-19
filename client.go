package mau

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hashicorp/mdns"
)

var (
	ErrFriendNotFollowed        = errors.New("Friend is not being followed.")
	ErrCantFindFriend           = errors.New("Couldn't find friend.")
	ErrIncorrectPeerCertificate = errors.New("Incorrect Peer certificate.")
)

type Client struct {
	http.Client
	account *Account
	peer    Fingerprint
}

// TODO(maybe) Cache clients map[Fingerprint]*Client
func (a *Account) Client(peer Fingerprint) (*Client, error) {
	cert, err := a.certificate()
	if err != nil {
		return nil, err
	}

	c := &Client{
		account: a,
		peer:    peer,
	}

	c.Client = http.Client{
		// Prevent Redirects
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse },
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
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
		},
	}

	return c, nil
}

func (c *Client) DownloadFriend(ctx context.Context, address string, fingerprint Fingerprint, after time.Time) error {
	followed := path.Join(c.account.path, fingerprint.String())
	if _, err := os.Stat(followed); err != nil {
		return ErrFriendNotFollowed
	}

	// if no address is provided we'll search for it with MDNS
	if address == "" {
		addresses := make(chan string, 1)
		go findFriend(ctx, fingerprint, addresses)
		select {
		case address = <-addresses:
		case <-ctx.Done():
			return ErrCantFindFriend
		}
		close(addresses)
	}

	// TODO if we still don't have an address lets search on the internet with Kad

	// Get list of remote files since the last modification we have
	url := fmt.Sprintf("%s/p2p/%s", address, fingerprint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	req.Header.Add("If-Modified-Since", after.UTC().Format(http.TimeFormat))

	resp, err := c.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Peer responded with error: %s", resp.Status)
	}

	var list []FileListItem
	err = json.NewDecoder(resp.Body).Decode(&list)
	resp.Body.Close()
	if err != nil {
		return err
	}

	// Download each file in order
	for i := range list {
		select {
		case <-ctx.Done():
			return nil
		default:
			err = c.DownloadFile(ctx, address, fingerprint, &list[i])
			if err != nil {
				log.Printf("Error: Downloading File %s\n\t%s", url, err)
			}
		}
	}

	return nil
}

func (c *Client) DownloadFile(ctx context.Context, address string, fingerprint Fingerprint, file *FileListItem) error {
	fpath := path.Join(c.account.path, fingerprint.String(), file.Name)

	f := File{
		Path:    fpath,
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

	url := fmt.Sprintf("%s/p2p/%s/%s", address, fingerprint, file.Name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Server returned unsuccessful response %s for %s", resp.Status, url)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if len(content) != int(file.Size) {
		return fmt.Errorf("Size is different for %s\nexpected %d received %d\ncontent:\n%s", url, file.Size, len(content), content)
	}

	hash := sha256.Sum256(content)
	h := fmt.Sprintf("%x", hash)
	if h != file.Sum {
		return fmt.Errorf("Hash sum is different received %s", h)
	}

	// TODO: check for file signature
	// TODO: check for file encrypted to current user
	// TODO: keep existing version

	err = os.WriteFile(fpath, content, 0600)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) verifyPeerCertificate(rawCerts [][]byte, _ [][]*x509.Certificate) error {
	return certIncludes(rawCerts, c.peer)
}

func findFriend(ctx context.Context, fingerprint Fingerprint, addresses chan<- string) error {
	name := fmt.Sprintf("%s.%s.%s.", fingerprint, mDNSServiceName, mDNSDomain)
	entriesCh := make(chan *mdns.ServiceEntry, cap(addresses))

	err := mdns.Lookup(mDNSServiceName, entriesCh)
	if err != nil {
		return err
	}

	for {
		select {
		case entry := <-entriesCh:
			if entry.Name == name {
				addresses <- fmt.Sprintf("%s://%s:%d", uriProtocolName, entry.AddrV4, entry.Port)
			}
		case <-ctx.Done():
			return nil
		}
	}
}
