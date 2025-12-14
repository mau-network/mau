package mau

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"io"
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
	Name string `json:"name"`
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
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
				},
				PreferServerCipherSuites: true,
				MinVersion:               tls.VersionTLS12,
				CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			},
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
		return err
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
	mdns_err := s.mdnsServer.Shutdown()
	http_err := s.httpServer.Close()
	s.dhtServer.Leave()

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
			Name: item.Name(),
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
	w.Write(marshaled)
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

	// TODO needs to support interrupt resume
	reader, err := os.Open(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
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

	file, err := s.account.GetFileVersion(fpr, segments[1], segments[2])
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

	// TODO needs to support interrupt resume
	reader, err := os.Open(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, reader)
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
