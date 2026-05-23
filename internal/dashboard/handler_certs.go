package dashboard

import (
	"crypto/x509"
	"encoding/pem"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type CertInfo struct {
	Domain    string    `json:"domain"`
	Issuer    string    `json:"issuer"`
	Subject   string    `json:"subject"`
	NotBefore time.Time `json:"not_before"`
	NotAfter  time.Time `json:"not_after"`
	IsExpired bool      `json:"is_expired"`
	DaysLeft  int       `json:"days_left"`
	FilePath  string    `json:"file_path"`
}

func (s *Server) handleListCerts(w http.ResponseWriter, r *http.Request) {
	certs := []CertInfo{}

	certPaths := map[string]string{}
	if s.cfg.Proxy.TLSCertFile != "" {
		certPaths[s.cfg.Proxy.TLSCertFile] = "proxy"
	}
	if s.cfg.Dashboard.TLSCertFile != "" {
		certPaths[s.cfg.Dashboard.TLSCertFile] = "dashboard"
	}

	certsDir := "./certs"
	if entries, err := os.ReadDir(certsDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() && (strings.HasSuffix(e.Name(), ".pem") || strings.HasSuffix(e.Name(), ".crt")) {
				certPaths[filepath.Join(certsDir, e.Name())] = "file"
			}
		}
	}

	for path := range certPaths {
		info := parseCertFile(path)
		if info != nil {
			certs = append(certs, *info)
		}
	}

	writeJSON(w, http.StatusOK, certs)
}

func parseCertFile(path string) *CertInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("cert: read %s: %v", path, err)
		return nil
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Printf("cert: parse %s: %v", path, err)
		return nil
	}

	domain := ""
	if len(cert.DNSNames) > 0 {
		domain = cert.DNSNames[0]
	} else {
		domain = cert.Subject.CommonName
	}

	now := time.Now()
	daysLeft := int(cert.NotAfter.Sub(now).Hours() / 24)

	return &CertInfo{
		Domain:    domain,
		Issuer:    cert.Issuer.CommonName,
		Subject:   cert.Subject.CommonName,
		NotBefore: cert.NotBefore,
		NotAfter:  cert.NotAfter,
		IsExpired: now.After(cert.NotAfter),
		DaysLeft:  daysLeft,
		FilePath:  path,
	}
}

func (s *Server) handleReloadCerts(w http.ResponseWriter, r *http.Request) {
	if s.OnCertReload != nil {
		s.OnCertReload()
	}
	s.recordAudit("admin", dashboardClientIP(r), "cert_reload", "success", "certificates reloaded")
	writeJSON(w, http.StatusOK, map[string]string{"message": "certificates reloaded"})
}
