package config

import "testing"

func TestDefaultConfigRedirectsHTTPToWAF(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Proxy.IPTablesPort != 80 {
		t.Fatalf("expected default iptables redirect port 80, got %d", cfg.Proxy.IPTablesPort)
	}
}

func TestValidateNormalizesIPTablesPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Proxy.IPTablesPort = 0
	if err := cfg.Validate(); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
	if cfg.Proxy.IPTablesPort != 80 {
		t.Fatalf("expected invalid iptables port to normalize to 80, got %d", cfg.Proxy.IPTablesPort)
	}
}
