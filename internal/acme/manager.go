package acme

import (
	"crypto/tls"
	"log"
	"net/http"

	"golang.org/x/crypto/acme/autocert"
)

type Manager struct {
	autocert.Manager
	enabled bool
}

// New creates an ACME manager for automatic TLS certificates.
// cacheDir is where certificates are stored. domains are the hostnames to manage.
func New(cacheDir string, email string, domains []string) *Manager {
	m := &Manager{
		enabled: len(domains) > 0,
		Manager: autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			Cache:      autocert.DirCache(cacheDir),
			HostPolicy: autocert.HostWhitelist(domains...),
		},
	}
	if email != "" {
		m.Manager.Email = email
	}
	return m
}

// Enabled returns whether ACME is active.
func (m *Manager) Enabled() bool {
	return m != nil && m.enabled
}

// TLSConfig returns a tls.Config that uses the ACME manager for certificates.
func (m *Manager) TLSConfig() *tls.Config {
	return &tls.Config{
		GetCertificate: m.Manager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
		MinVersion:     tls.VersionTLS12,
	}
}

// HTTPHandler returns an HTTP handler for ACME HTTP-01 challenges.
// Mount this on port 80 or on /.well-known/acme-challenge/
func (m *Manager) HTTPHandler(fallback http.Handler) http.Handler {
	return m.Manager.HTTPHandler(fallback)
}

// GetCertificate wraps autocert.Manager.GetCertificate with logging.
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cert, err := m.Manager.GetCertificate(hello)
	if err != nil {
		log.Printf("ACME: failed to get certificate for %s: %v", hello.ServerName, err)
	}
	return cert, err
}
