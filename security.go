package main

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"log"
)

func IsNodeCert(id DHTKey, connection *tls.ConnectionState) bool {
	if connection == nil {
		return false
	}

	for _, c := range connection.PeerCertificates {
		switch c.PublicKeyAlgorithm {
		case x509.RSA:
			if pubkey, ok := c.PublicKey.(rsa.PublicKey); ok {
				return pubkey.Equal(id)
			}
		case x509.ECDSA:
			if pubkey, ok := c.PublicKey.(ecdsa.PublicKey); ok {
				return pubkey.Equal(id)
			}
		case x509.Ed25519:
			if pubkey, ok := c.PublicKey.(ed25519.PublicKey); ok {
				return pubkey.Equal(id)
			}

		default:
			log.Printf("Error public key algorithm not supported %s", c.PublicKeyAlgorithm)
		}
	}

	return false
}
