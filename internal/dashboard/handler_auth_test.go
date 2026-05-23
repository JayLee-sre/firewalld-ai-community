package dashboard

import (
	"net/http"
	"testing"
)

func TestDashboardClientIPIgnoresSpoofedForwardedHeader(t *testing.T) {
	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "203.0.113.10:45678"
	req.Header.Set("X-Forwarded-For", "8.8.8.8")

	if got := dashboardClientIP(req); got != "203.0.113.10" {
		t.Fatalf("expected direct peer IP, got %q", got)
	}
}

func TestDashboardClientIPTrustsLocalProxyHeader(t *testing.T) {
	req, err := http.NewRequest("POST", "/api/v1/login", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.RemoteAddr = "127.0.0.1:45678"
	req.Header.Set("X-Forwarded-For", "8.8.8.8, 127.0.0.1")

	if got := dashboardClientIP(req); got != "8.8.8.8" {
		t.Fatalf("expected forwarded client IP, got %q", got)
	}
}
