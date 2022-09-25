package mau

import (
	"context"
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
	account        *Account
	httpServer     http.Server
	mdnsServer     *mdns.Server
	dhtServer      *dhtServer
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

	router := mux.NewRouter()
	s := Server{
		account:        a,
		resultsLimit:   100,
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

	router.HandleFunc("/p2p/{FPR:[0-9a-f]+}", s.list).Methods("GET")
	router.HandleFunc("/p2p/{FPR:[0-9a-f]+}/{fileID}", s.get).Methods("GET")
	router.HandleFunc("/p2p/{FPR:[0-9a-f]+}/{fileID}/{versionID}", s.version).Methods("GET")
	router.PathPrefix("/kad/").Handler(s.dhtServer)

	return &s, nil
}

func (s *Server) Serve(l net.Listener) error {
	port := l.Addr().(*net.TCPAddr).Port

	if err := s.serveMDNS(port); err != nil {
		return err
	}

	if len(s.bootstrapNodes) > 0 {
		if err := s.serveDHT(port); err != nil {
			return err
		}
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

// TODO improve this method to take a context and be cancellable along with serveMDNS and serve methods
func (s *Server) serveDHT(port int) error {
	upnp, err := newUPNPClient(context.Background())
	if err != nil {
		return err
	}

	err = upnp.AddPortMapping("", uint16(port), "tcp", uint16(port), "", true, "Mau p2p service", 0)
	if err != nil {
		return err
	}

	externalAddress, err := upnp.GetExternalIPAddress()
	if err != nil {
		return err
	}

	s.dhtServer = newDHTRPC(s.account, fmt.Sprintf("%s:%d", externalAddress, port))
	s.dhtServer.Join(s.bootstrapNodes)
	return nil
}

func (s *Server) Close() error {
	// TODO check why the fuck this panics when closing the server?
	// regardless of the errors I need to try closing all interfaces
	mdns_err := s.mdnsServer.Shutdown()
	http_err := s.httpServer.Close()
	s.dhtServer.Leave()

	if mdns_err != nil {
		return mdns_err
	}

	return http_err
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince == "" {
		http.Error(w, "Missing If-Modified-Since header", http.StatusBadRequest)
		return
	}

	lastModified, err := http.ParseTime(ifModifiedSince)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	fprStr := vars["FPR"]

	var fpr Fingerprint

	fpr, err = ParseFingerprint(fprStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	page := s.account.ListFiles(fpr, lastModified, s.resultsLimit)

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
	vars := mux.Vars(r)

	fprStr := vars["FPR"]
	var fpr Fingerprint
	var err error
	fpr, err = ParseFingerprint(fprStr)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := s.account.GetFile(fpr, vars["fileID"])
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

	fpr, err := ParseFingerprint(vars["FPR"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	file, err := s.account.GetFileVersion(fpr, vars["fileID"], vars["versionID"])
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

func isPermitted(certs []*x509.Certificate, recipients []*Friend) bool {
	for _, c := range certs {
		var id Fingerprint
		switch c.PublicKeyAlgorithm {
		case x509.RSA:
			pubkey := c.PublicKey.(*rsa.PublicKey)
			id = packet.NewRSAPublicKey(c.NotBefore, pubkey).Fingerprint
		default:
			fmt.Println("Error public key algorithm not supported")
			return false
		}

		for _, r := range recipients {
			if id == r.Fingerprint() {
				return true
			}
		}
	}

	return false
}
