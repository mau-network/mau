package mau

var (
	rsaKeyLength = 4096
)

const (
	serverResultLimit  = 100
	mDNSServiceName    = "_mau._tcp"
	mauDirName         = ".mau"
	accountKeyFilename = "account.pgp"
	dirPerm            = 0700
	uriProtocolName    = "https"
	mDNSDomain         = "local"
)
