// Package devcert generates ephemeral TLS certificates for local development.
// The certificates use ECDSA P-256 and are valid for 10 days, matching the
// pattern from quic-go/webtransport-go interop tests.
package devcert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"
	"time"
)

// Generate creates an ephemeral ECDSA P-256 self-signed certificate valid for
// 10 days. It returns the tls.Certificate and the SHA-256 hash of the
// DER-encoded certificate.
func Generate() (tls.Certificate, [32]byte, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return tls.Certificate{}, [32]byte{}, err
	}
	serial := int64(binary.BigEndian.Uint64(b))
	if serial < 0 {
		serial = -serial
	}

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, [32]byte{}, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(serial),
		Subject:               pkix.Name{},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 24 * time.Hour),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, [32]byte{}, err
	}

	leaf, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, [32]byte{}, err
	}

	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
		Leaf:        leaf,
	}
	hash := sha256.Sum256(leaf.Raw)

	return cert, hash, nil
}

// FormatHashJS formats a SHA-256 cert hash as a JS array literal: [0x12, 0x34, ...]
func FormatHashJS(hash [32]byte) string {
	s := strings.ReplaceAll(fmt.Sprintf("%#v", hash[:]), "[]byte{", "[")
	return strings.ReplaceAll(s, "}", "]")
}
