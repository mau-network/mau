package mau

import "time"

var (
	rsaKeyLength = 4096
)

const (
	serverResultLimit  = 100
	mDNSServiceName    = "_mau._tcp"
	mauDirName         = ".mau"
	accountKeyFilename = "account.pgp"
	syncStateFilename  = "sync_state.json"
	DirPerm            = 0700
	FilePerm           = 0600
	uriProtocolName    = "https"
	mDNSDomain         = "local"
	httpClientTimeout  = 3 * time.Second
)
