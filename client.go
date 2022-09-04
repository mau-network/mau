package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/hashicorp/mdns"
	"github.com/multiformats/go-multihash"
)

type Client struct {
	account    *Account
	httpClient *http.Client
}

const (
	URIProtocolName = "https"
	MDNSDomain      = "local"
)

func NewClient(account *Account) (*Client, error) {
	cert, err := account.Certificate()
	if err != nil {
		return nil, err
	}

	c := Client{
		account: account,
		httpClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       []tls.Certificate{*cert},
					InsecureSkipVerify: true,
					CipherSuites: []uint16{
						tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
						tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
						tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					},
				},
			},
		},
	}

	return &c, nil
}

func FindFriend(ctx context.Context, fingerprint string, addresses chan<- string) error {
	defer close(addresses)

	name := fmt.Sprintf("%s.%s.%s.", fingerprint, MDNSServiceName, MDNSDomain)
	entriesCh := make(chan *mdns.ServiceEntry, cap(addresses))

	err := mdns.Lookup(MDNSServiceName, entriesCh)
	if err != nil {
		return err
	}

	for {
		select {
		case entry := <-entriesCh:
			if entry.Name == name {
				addresses <- fmt.Sprintf("%s://%s:%d", URIProtocolName, entry.AddrV4, entry.Port)
			}
		case <-ctx.Done():
			return nil
		}
	}

	return nil
}

func DownloadFriend(ctx context.Context, account *Account, address, fingerprint string, after time.Time, client *Client) error {
	// Get list of remote files since the last modification we have
	url := fmt.Sprintf("%s/p2p/%s", address, fingerprint)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	utc, err := time.LoadLocation("UTC")
	if err != nil {
		return err
	}

	req.Header.Add("If-Modified-Since", after.In(utc).Format(http.TimeFormat))

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return err
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
			err = DownloadFile(ctx, account, address, fingerprint, &list[i], client)
			if err != nil {
				log.Printf("Error: Downloading File %s\n\t%s", url, err)
			}
		}
	}

	return nil
}

func DownloadFile(ctx context.Context, account *Account, address, fingerprint string, file *FileListItem, client *Client) error {
	fpath := path.Join(account.path, fingerprint, file.Name)

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

	resp, err := client.httpClient.Do(req)
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

	hash, err := multihash.Sum(content, multihash.SHA2_256, -1)
	if err != nil {
		return err
	}

	h := hash.B58String()
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
