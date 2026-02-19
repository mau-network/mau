package mau

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/mdns"
)

type Server struct {
	account *Account

	router *http.ServeMux

	httpServer http.Server
	mdnsServer *mdns.Server
	dhtServer  *dhtServer

	bootstrapNodes []*Peer
	resultsLimit   uint
}

type FileListItem struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
	Sum  string `json:"sum"`
}

func (a *Account) Server(knownNodes []*Peer) (*Server, error) {
	cert, err := a.certificate(nil)
	if err != nil {
		return nil, err
	}

	router := http.NewServeMux()

	s := Server{
		account:        a,
		router:         router,
		resultsLimit:   serverResultLimit,
		bootstrapNodes: knownNodes,
		httpServer: http.Server{
			Handler: router,
			TLSConfig: &tls.Config{
				Certificates:       []tls.Certificate{cert},
				InsecureSkipVerify: true,
				ClientAuth:         tls.RequestClientCert,
				// Go 1.26: Use secure defaults, removed deprecated PreferServerCipherSuites
				// and explicit CipherSuites (modern Go chooses optimal suites automatically)
				MinVersion: tls.VersionTLS13, // TLS 1.3 for better security and performance
				CurvePreferences: []tls.CurveID{
					tls.X25519, // Modern, fast elliptic curve
					tls.CurveP256,
					tls.CurveP384,
					tls.CurveP521,
				},
			},
			ReadTimeout:       30 * time.Second,
			WriteTimeout:      30 * time.Second,
			IdleTimeout:       120 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}

	router.Handle("/p2p/", &s)

	return &s, nil
}

var (
	listReg    = regexp.MustCompile(`^/p2p/[0-9a-f]+$`)
	getReg     = regexp.MustCompile(`^/p2p/[0-9a-f]+/([^/]+)$`)
	versionReg = regexp.MustCompile(`^/p2p/[0-9a-f]+/([^/]+).version/([^/]+)$`)
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "", http.StatusMethodNotAllowed)
	}

	switch {
	case listReg.MatchString(r.URL.Path):
		s.list(w, r)
	case getReg.MatchString(r.URL.Path):
		s.get(w, r)
	case versionReg.MatchString(r.URL.Path):
		s.version(w, r)
	default:
		http.Error(w, "", http.StatusBadRequest)
	}
}

func (s *Server) Serve(l net.Listener, externalAddress string) error {
	if l == nil || l.Addr() == nil {
		return errors.New("listener cannot be nil")
	}
	
	port := l.Addr().(*net.TCPAddr).Port

	if err := s.serveMDNS(port); err != nil {
		return err
	}

	if err := s.serveDHT(context.Background(), externalAddress); err != nil {
		return err
	}

	return s.httpServer.ServeTLS(l, "", "")
}

func (s *Server) serveMDNS(port int) error {
	fingerprint := s.account.Fingerprint().String()

	service, err := mdns.NewMDNSService(fingerprint, mDNSServiceName, "", "", port, nil, []string{})
	if err != nil {
		return err
	}

	server, err := mdns.NewServer(&mdns.Config{Zone: service})
	if err != nil {
		// MDNS might fail when IPv6 is disabled - log but don't fail entirely
		// Discovery will still work via DHT and manual addresses
		slog.Warn("Failed to start mDNS server", "error", err)
		return nil
	}

	s.mdnsServer = server

	return nil
}

func (s *Server) serveDHT(ctx context.Context, externalAddress string) error {
	s.dhtServer = newDHTServer(s.account, externalAddress)
	s.router.Handle("/kad/", s.dhtServer)
	s.dhtServer.Join(ctx, s.bootstrapNodes)
	return nil
}

func (s *Server) Close() error {
	var mdns_err error
	if s.mdnsServer != nil {
		mdns_err = s.mdnsServer.Shutdown()
	}
	http_err := s.httpServer.Close()
	if s.dhtServer != nil {
		s.dhtServer.Leave()
	}

	return errors.Join(mdns_err, http_err)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	var lastModified time.Time
	var err error

	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" {

		lastModified, err = http.ParseTime(ifModifiedSince)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}

	fprStr := strings.TrimPrefix(r.URL.Path, "/p2p/")
	fpr, err := FingerprintFromString(fprStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page := s.account.ListFiles(fpr, lastModified, s.resultsLimit)

	list := make([]FileListItem, 0, len(page))
	for _, item := range page {
		hash, err := item.Hash()
		if err != nil {
			slog.Error("failed to calculate file hash", "file", item.Name(), "error", err)
			hash = ""
		}

		size, err := item.Size()
		if err != nil {
			slog.Error("failed to calculate file size", "file", item.Name(), "error", err)
			hash = ""
		}

		recipients, err := item.Recipients(s.account)
		if err != nil {
			continue
		}

		permitted := isPermitted(r.TLS.PeerCertificates, recipients)
		if !permitted {
			continue
		}

		list = append(list, FileListItem{
			Path: "/p2p/" + fpr.String() + "/" + item.Name(),
			Size: size,
			Sum:  hash,
		})
	}

	marshaled, err := json.Marshal(list)
	if err != nil {
		http.Error(w, "Error while processing the list of files", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(marshaled); err != nil {
		http.Error(w, "Error writing response", http.StatusInternalServerError)
	}
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/p2p/"), "/")
	if len(segments) != 2 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	fprStr := segments[0]
	var fpr Fingerprint
	var err error
	fpr, err = FingerprintFromString(fprStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := s.account.GetFile(fpr, segments[1])
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	recipients, err := file.Recipients(s.account)
	if err != nil {
		http.Error(w, "Error reading file recipients", http.StatusInternalServerError)
		return
	}

	allowed := isPermitted(r.TLS.PeerCertificates, recipients)
	if !allowed {
		http.Error(w, "Error file is not allowed for user", http.StatusUnauthorized)
		return
	}

	reader, err := os.Open(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Accept-Ranges", "bytes")
	http.ServeContent(w, r, file.Name(), time.Time{}, reader)
}

func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/p2p/"), "/")
	if len(segments) != 3 {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	fpr, err := FingerprintFromString(segments[0])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Strip .version suffix from filename
	// URL format: /p2p/{fpr}/{filename}.version/{hash}
	// But GetFileVersion expects just {filename}
	filename := strings.TrimSuffix(segments[1], ".version")

	file, err := s.account.GetFileVersion(fpr, filename, segments[2])
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	recipients, err := file.Recipients(s.account)
	if err != nil {
		http.Error(w, "Error reading file recipients", http.StatusInternalServerError)
		return
	}

	allowed := isPermitted(r.TLS.PeerCertificates, recipients)
	if !allowed {
		http.Error(w, "Error file is not allowed for user", http.StatusUnauthorized)
		return
	}

	reader, err := os.Open(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}
	defer reader.Close()

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Accept-Ranges", "bytes")
	http.ServeContent(w, r, file.Name(), time.Time{}, reader)
}

func isPermitted(certs []*x509.Certificate, recipients []*Friend) bool {
	fpr, err := FingerprintFromCert(certs)
	if err != nil {
		return false
	}

	for _, r := range recipients {
		if fpr == r.Fingerprint() {
			return true
		}
	}

	return false
}
