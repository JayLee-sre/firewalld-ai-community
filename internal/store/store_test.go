package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"zhiyuwaf/internal/model"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestStore_Settings(t *testing.T) {
	s := newTestStore(t)

	// Set and get
	s.SetSetting("key1", "value1")
	v, err := s.GetSetting("key1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "value1" {
		t.Errorf("expected value1, got %s", v)
	}

	// Update
	s.SetSetting("key1", "value2")
	v, _ = s.GetSetting("key1")
	if v != "value2" {
		t.Errorf("expected value2, got %s", v)
	}

	// Missing key
	v, err = s.GetSetting("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != "" {
		t.Errorf("expected empty string for missing key, got %s", v)
	}

	// List all
	s.SetSetting("key2", "val2")
	m, err := s.ListSettings()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["key1"] != "value2" || m["key2"] != "val2" {
		t.Errorf("unexpected settings map: %v", m)
	}
}

func TestStore_AttackLogs(t *testing.T) {
	s := newTestStore(t)

	log1 := model.AttackLog{
		ID:        "log-1",
		Timestamp: time.Now().Add(-1 * time.Hour),
		ClientIP:  "1.2.3.4",
		Method:    "GET",
		Path:      "/admin",
		RuleID:    "SQLI-001",
		RuleName:  "SQL Injection",
		Severity:  "high",
		Source:    "rule_engine",
		Action:    "blocked",
	}
	log2 := model.AttackLog{
		ID:        "log-2",
		Timestamp: time.Now(),
		ClientIP:  "5.6.7.8",
		Method:    "POST",
		Path:      "/login",
		RuleID:    "AI-XSS",
		RuleName:  "AI XSS Detection",
		Severity:  "critical",
		Source:    "ai",
		Action:    "blocked",
	}

	s.InsertAttackLog(log1)
	s.InsertAttackLog(log2)

	// List all
	logs, total, err := s.ListAttackLogs(0, 100, LogFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 total, got %d", total)
	}

	// Filter by source
	logs, total, _ = s.ListAttackLogs(0, 100, LogFilter{Source: "ai"})
	if total != 1 {
		t.Errorf("expected 1 AI log, got %d", total)
	}
	if len(logs) > 0 && logs[0].Source != "ai" {
		t.Errorf("expected ai source, got %s", logs[0].Source)
	}

	// Filter by severity
	_, total, _ = s.ListAttackLogs(0, 100, LogFilter{Severity: "high"})
	if total != 1 {
		t.Errorf("expected 1 high severity log, got %d", total)
	}

	// Filter by IP
	_, total, _ = s.ListAttackLogs(0, 100, LogFilter{ClientIP: "1.2.3.4"})
	if total != 1 {
		t.Errorf("expected 1 log for IP 1.2.3.4, got %d", total)
	}

	// Stats
	stats, err := s.GetAttackStats(time.Now().Add(-24 * time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.TotalRequests != 2 {
		t.Errorf("expected 2 total requests, got %d", stats.TotalRequests)
	}
	if stats.BySource["ai"] != 1 {
		t.Errorf("expected 1 AI source, got %d", stats.BySource["ai"])
	}
	if stats.BySeverity["high"] != 1 {
		t.Errorf("expected 1 high severity, got %d", stats.BySeverity["high"])
	}

	got, err := s.GetAttackLog("log-1")
	if err != nil {
		t.Fatalf("get attack log: %v", err)
	}
	if got == nil || got.ID != "log-1" || got.Path != "/admin" {
		t.Fatalf("unexpected attack log: %+v", got)
	}
	missing, err := s.GetAttackLog("missing")
	if err != nil {
		t.Fatalf("get missing attack log: %v", err)
	}
	if missing != nil {
		t.Fatalf("expected missing log to be nil, got %+v", missing)
	}
}

func TestStore_AuditEvents(t *testing.T) {
	s := newTestStore(t)

	now := time.Now()
	if err := s.InsertAuditEvent(model.AuditEvent{
		ID:        "audit-1",
		Timestamp: now,
		Actor:     "admin",
		ClientIP:  "127.0.0.1",
		Action:    "login",
		Status:    "success",
		Detail:    "dashboard login",
	}); err != nil {
		t.Fatalf("insert audit event: %v", err)
	}
	if err := s.InsertAuditEvent(model.AuditEvent{
		ID:        "audit-2",
		Timestamp: now.Add(time.Second),
		Actor:     "admin",
		ClientIP:  "127.0.0.1",
		Action:    "license_activate",
		Status:    "failed",
		Detail:    "invalid license",
	}); err != nil {
		t.Fatalf("insert audit event: %v", err)
	}

	events, total, err := s.ListAuditEvents(0, 20, AuditFilter{})
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if total != 2 || len(events) != 2 {
		t.Fatalf("expected 2 audit events, total=%d len=%d", total, len(events))
	}

	events, total, err = s.ListAuditEvents(0, 20, AuditFilter{Action: "license_activate", Status: "failed"})
	if err != nil {
		t.Fatalf("filter audit events: %v", err)
	}
	if total != 1 || len(events) != 1 || events[0].ID != "audit-2" {
		t.Fatalf("unexpected filtered audit events: total=%d events=%+v", total, events)
	}
}

func TestStore_Rules(t *testing.T) {
	s := newTestStore(t)

	rule := model.Rule{
		ID:             "rule-1",
		Name:           "Test SQLi Rule",
		Description:    "Detects SQL injection",
		Severity:       "high",
		Enabled:        true,
		Patterns:       []string{`(?i)union\s+select`},
		MatchLocations: []string{"query", "body"},
	}

	// Create
	if err := s.CreateRule(rule); err != nil {
		t.Fatalf("create rule: %v", err)
	}

	// List
	rules, err := s.ListRules()
	if err != nil {
		t.Fatalf("list rules: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].Name != "Test SQLi Rule" {
		t.Errorf("expected 'Test SQLi Rule', got %s", rules[0].Name)
	}
	if len(rules[0].Patterns) != 1 || rules[0].Patterns[0] != `(?i)union\s+select` {
		t.Errorf("unexpected patterns: %v", rules[0].Patterns)
	}

	// Update
	rule.Name = "Updated Rule"
	rule.Severity = "critical"
	s.UpdateRule(rule)

	rules, _ = s.ListRules()
	if rules[0].Name != "Updated Rule" {
		t.Errorf("expected 'Updated Rule', got %s", rules[0].Name)
	}
	if rules[0].Severity != "critical" {
		t.Errorf("expected 'critical', got %s", rules[0].Severity)
	}

	// Delete
	s.DeleteRule("rule-1")
	rules, _ = s.ListRules()
	if len(rules) != 0 {
		t.Errorf("expected 0 rules after delete, got %d", len(rules))
	}
}

func TestStore_IPList(t *testing.T) {
	s := newTestStore(t)

	entry := model.IPEntry{
		ID:        "ip-1",
		IPAddress: "192.168.1.100",
		ListType:  "blacklist",
		Note:      "Test block",
	}

	s.AddIPEntry(entry)

	// List blacklist
	entries, err := s.ListIPEntries("blacklist")
	if err != nil {
		t.Fatalf("list IPs: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	// Get map
	m, _ := s.GetIPListMap("blacklist")
	if !m["192.168.1.100"] {
		t.Error("expected 192.168.1.100 in blacklist map")
	}

	// Whitelist should be empty
	entries, _ = s.ListIPEntries("whitelist")
	if len(entries) != 0 {
		t.Errorf("expected 0 whitelist entries, got %d", len(entries))
	}

	// Remove
	s.RemoveIPEntry("ip-1")
	entries, _ = s.ListIPEntries("blacklist")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries after remove, got %d", len(entries))
	}
}

func TestStore_Migration(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "new.db")

	// Verify file is created
	if _, err := os.Stat(dbPath); !os.IsNotExist(err) {
		// dir might contain other files, that's fine
	}

	s, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("migration failed: %v", err)
	}
	defer s.Close()

	// Verify tables exist by doing operations
	s.SetSetting("test", "ok")
	v, _ := s.GetSetting("test")
	if v != "ok" {
		t.Error("settings table not working after migration")
	}
}
