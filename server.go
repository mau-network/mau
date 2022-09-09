package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	// TODO: Remove dependency
	"github.com/gorilla/mux"
	"github.com/hashicorp/mdns"
	"golang.org/x/crypto/openpgp/packet"
)

type Server struct {
	account    *Account
	httpServer http.Server
	mdnsServer *mdns.Server
	limit      uint
}

type FileListItem struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Sum  string `json:"sum"`
}

// TODO: Change it to UDP over QUIC protocol
func NewServer(account *Account) (*Server, error) {
	cert, err := account.Certificate()
	if err != nil {
		return nil, err
	}

	router := mux.NewRouter()
	s := Server{
		account: account,
		limit:   100,
		httpServer: http.Server{
			Handler: router,
			TLSConfig: &tls.Config{
				Certificates:       []tls.Certificate{*cert},
				InsecureSkipVerify: true,
				ClientAuth:         tls.RequestClientCert,
				CipherSuites: []uint16{
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
				PreferServerCipherSuites: true,
				MinVersion:               tls.VersionTLS12,
				CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
			},
			TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler), 0),
		},
	}

	router.HandleFunc("/p2p/{FPR:[0-9A-F]+}", s.list).Methods("GET")
	router.HandleFunc("/p2p/{FPR:[0-9A-F]+}/{fileID}", s.get).Methods("GET")
	router.HandleFunc("/p2p/{FPR:[0-9A-F]+}/{fileID}/{versionID}", s.version).Methods("GET")

	return &s, nil
}

func (s *Server) Serve(l net.Listener) error {
	port := l.Addr().(*net.TCPAddr).Port

	if err := s.serveMDNS(port); err != nil {
		return err
	}

	return s.httpServer.ServeTLS(l, "", "")
}

func (s *Server) serveMDNS(port int) error {
	fingerprint := s.account.Fingerprint()

	service, err := mdns.NewMDNSService(fingerprint, MDNSServiceName, "", "", port, nil, []string{})
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

func (s *Server) Close() error {
	// regardless of the errors I need to try closing all interfaces
	mdns_err := s.mdnsServer.Shutdown()
	http_err := s.httpServer.Close()

	if mdns_err != nil {
		return mdns_err
	}

	return http_err
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince == "" {
		http.Error(w, "Missing If-Modified_Since header", http.StatusBadRequest)
		return
	}

	lastModified, err := http.ParseTime(ifModifiedSince)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	page := ListFiles(s.account, vars["FPR"], lastModified, s.limit)

	list := make([]FileListItem, 0, len(page))
	for _, item := range page {
		hash, err := item.Hash()
		if err != nil {
			fmt.Println("There was an error calculating hash: " + err.Error())
			hash = ""
		}

		size, err := item.Size()
		if err != nil {
			fmt.Println("There was a error calculating size: " + err.Error())
			hash = ""
		}

		recepients, err := item.Recipients(s.account)
		if err != nil {
			continue
		}

		permitted := isPermitted(r.TLS.PeerCertificates, recepients)
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
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(marshaled)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	file, err := GetFile(s.account, vars["FPR"], vars["fileID"])
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	recepients, err := file.Recipients(s.account)
	if err != nil {
		http.Error(w, "Error reading file recepients", http.StatusInternalServerError)
		return
	}

	allowed := isPermitted(r.TLS.PeerCertificates, recepients)
	if !allowed {
		http.Error(w, "Error file is not allowed for user", http.StatusUnauthorized)
		return
	}

	// TODO needs to support interrupt resume
	content, err := os.ReadFile(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Length", fmt.Sprint(len(content)))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (s *Server) version(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	file, err := GetVersion(s.account, vars["FPR"], vars["fileID"], vars["versionID"])
	if err != nil {
		http.Error(w, "File not found", http.StatusNotFound)
		return
	}

	recepients, err := file.Recipients(s.account)
	if err != nil {
		http.Error(w, "Error reading file recepients", http.StatusInternalServerError)
		return
	}

	allowed := isPermitted(r.TLS.PeerCertificates, recepients)
	if !allowed {
		http.Error(w, "Error file is not allowed for user", http.StatusUnauthorized)
		return
	}

	// TODO needs to support interrupt resume
	content, err := os.ReadFile(file.Path)
	if err != nil {
		http.Error(w, "Error reading file", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Length", fmt.Sprint(len(content)))
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func isPermitted(certs []*x509.Certificate, recepients []*Friend) bool {
	if len(recepients) == 0 {
		return true
	}

	for _, c := range certs {
		var id string
		switch c.PublicKeyAlgorithm {
		case x509.RSA:
			pubkey := c.PublicKey.(*rsa.PublicKey)
			id = fmt.Sprintf("%X", packet.NewRSAPublicKey(c.NotBefore, pubkey).Fingerprint)
		default:
			fmt.Println("Error public key algorithm not supported")
			return false
		}

		for _, r := range recepients {
			if id == r.Fingerprint() {
				return true
			}
		}
	}

	return false
}
