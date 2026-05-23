package dashboard

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/license"
	"zhiyuwaf/internal/store"
)

func newTestDashboardServer(t *testing.T) *Server {
	t.Helper()
	s, err := store.NewStore(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	cfg := config.DefaultConfig()
	cfg.Dashboard.JWTSecret = "test-secret"
	return NewServer(cfg, s)
}

func TestCurrentEditionRequiresSignedLicenseToken(t *testing.T) {
	srv := newTestDashboardServer(t)

	if got := srv.currentEdition(); got != "community" {
		t.Fatalf("empty license should be community, got %q", got)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	srv.cfg.License.PublicKey = base64.RawURLEncoding.EncodeToString(pub)
	payload := license.Payload{
		LicenseID:   "lic_test",
		Edition:     "pro",
		Customer:    "test",
		MachineID:   license.MachineID(),
		Features:    []string{"ai", "audit", "ssh_guard"},
		ExpiresAt:   time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		IssuedAt:    time.Now().Unix(),
		NextCheckAt: time.Now().Add(time.Hour).Unix(),
		GraceUntil:  time.Now().Add(24 * time.Hour).Unix(),
		Status:      "active",
	}
	data, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, data)
	token := base64.RawURLEncoding.EncodeToString(data) + "." + base64.RawURLEncoding.EncodeToString(sig)
	if err := srv.store.SetSetting("license_token", token); err != nil {
		t.Fatalf("set token: %v", err)
	}
	if got := srv.currentEdition(); got != "pro" {
		t.Fatalf("valid signed token should enable pro, got %q", got)
	}
}
