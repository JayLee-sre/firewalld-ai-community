package proxy

import "testing"

func TestRememberRedirectedPortDeduplicates(t *testing.T) {
	m := NewIPTablesManager(8080, true)
	m.rememberRedirectedPort(80)
	m.rememberRedirectedPort(80)
	m.rememberRedirectedPort(443)

	if len(m.redirectedPorts) != 2 {
		t.Fatalf("expected 2 redirected ports, got %d", len(m.redirectedPorts))
	}
	if m.redirectedPorts[0] != 80 || m.redirectedPorts[1] != 443 {
		t.Fatalf("unexpected redirected ports: %+v", m.redirectedPorts)
	}
}
