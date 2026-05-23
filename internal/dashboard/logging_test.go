package dashboard

import (
	"net/url"
	"testing"
)

func TestRedactedRequestURIHidesSensitiveQueryValues(t *testing.T) {
	u, err := url.Parse("/api/v1/logs/stream?token=abc123&limit=20&api_key=secret")
	if err != nil {
		t.Fatal(err)
	}

	got := redactedRequestURI(u)
	if got != "/api/v1/logs/stream?api_key=%5BREDACTED%5D&limit=20&token=%5BREDACTED%5D" {
		t.Fatalf("unexpected redacted URI: %q", got)
	}
}
