package sshmon

import (
	"net"
	"path/filepath"
	"testing"

	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/store"
)

func TestProcessLineAcceptedLoginDoesNotPanic(t *testing.T) {
	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, nil, nil)

	m.processLine("May 15 10:00:00 host sshd[123]: Accepted password for root from 203.0.113.10 port 55222 ssh2")
}

func TestIPTablesBinaryForIP(t *testing.T) {
	if got := iptablesBinaryForIP(net.ParseIP("203.0.113.10")); got != "iptables" {
		t.Fatalf("expected iptables for IPv4, got %q", got)
	}
	if got := iptablesBinaryForIP(net.ParseIP("2001:db8::1")); got != "ip6tables" {
		t.Fatalf("expected ip6tables for IPv6, got %q", got)
	}
}

func TestBlockIPSkipsInvalidAndLocalAddresses(t *testing.T) {
	m := New(Config{Enabled: true, MaxFails: 1, BanMinutes: 1, IPTablesEnabled: true}, nil, nil)

	m.blockIP("not-an-ip", "")
	m.blockIP("127.0.0.1", "")
	m.blockIP("::1", "")
}

func TestStopIsIdempotent(t *testing.T) {
	m := New(Config{Enabled: true, MaxFails: 1, BanMinutes: 1}, nil, nil)

	m.Stop()
	m.Stop()
}

func TestSuccessfulLoginFromWhitelistIsNotLogged(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.AddIPEntry(model.IPEntry{ID: "wl-1", IPAddress: "203.0.113.10", ListType: "whitelist"}); err != nil {
		t.Fatalf("add whitelist: %v", err)
	}

	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, s, nil)
	m.processLine("May 15 10:00:00 host sshd[123]: Accepted password for root from 203.0.113.10 port 55222 ssh2")

	events, total, err := s.ListSSHEvents(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if total != 0 || len(events) != 0 {
		t.Fatalf("expected no success event for whitelisted IP, got total=%d events=%d", total, len(events))
	}
}

func TestFailedLoginFromWhitelistIsLogged(t *testing.T) {
	s := newTestStore(t)
	defer s.Close()

	if err := s.AddIPEntry(model.IPEntry{ID: "wl-1", IPAddress: "203.0.113.10", ListType: "whitelist"}); err != nil {
		t.Fatalf("add whitelist: %v", err)
	}

	m := New(Config{Enabled: true, MaxFails: 3, BanMinutes: 1}, s, nil)
	m.processLine("May 15 10:00:00 host sshd[123]: Failed password for root from 203.0.113.10 port 55222 ssh2")

	events, total, err := s.ListSSHEvents(0, 10, "", "", "")
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if total != 1 || len(events) != 1 {
		t.Fatalf("expected failed event for whitelisted IP, got total=%d events=%d", total, len(events))
	}
	if events[0].EventType != "failed" {
		t.Fatalf("expected failed event, got %q", events[0].EventType)
	}
}

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.NewStore(filepath.Join(t.TempDir(), "zhiyu-waf.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	return s
}
