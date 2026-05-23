package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(dsn string, maxOpen, maxIdle int) (*MySQLStore, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}
	if maxOpen <= 0 {
		maxOpen = 25
	}
	if maxIdle <= 0 {
		maxIdle = 10
	}
	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	s := &MySQLStore{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

func (s *MySQLStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *MySQLStore) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS attack_logs (
			id             VARCHAR(36) PRIMARY KEY,
			timestamp      DATETIME NOT NULL,
			client_ip      VARCHAR(45) NOT NULL,
			region         VARCHAR(100) DEFAULT '',
			method         VARCHAR(10) NOT NULL,
			path           TEXT NOT NULL,
			headers        TEXT,
			body_preview   TEXT,
			rule_id        VARCHAR(100),
			rule_name      VARCHAR(255),
			severity       VARCHAR(20),
			source         VARCHAR(20),
			action         VARCHAR(20) DEFAULT 'blocked',
			ai_reasoning   TEXT,
			reviewed       TINYINT(1) DEFAULT 0,
			false_positive TINYINT(1) DEFAULT 0,
			site_id        VARCHAR(36) DEFAULT '',
			site_name      VARCHAR(255) DEFAULT '',
			domain         VARCHAR(255) DEFAULT '',
			INDEX idx_attack_logs_timestamp (timestamp),
			INDEX idx_attack_logs_client_ip (client_ip),
			INDEX idx_attack_logs_severity (severity)
		)`,
		`CREATE TABLE IF NOT EXISTS ssh_events (
			id         VARCHAR(36) PRIMARY KEY,
			timestamp  DATETIME NOT NULL,
			client_ip  VARCHAR(45) NOT NULL,
			region     VARCHAR(100) DEFAULT '',
			username   VARCHAR(100),
			event_type VARCHAR(20) NOT NULL,
			message    TEXT,
			INDEX idx_ssh_events_timestamp (timestamp),
			INDEX idx_ssh_events_client_ip (client_ip)
		)`,
		`CREATE TABLE IF NOT EXISTS rules (
			id              VARCHAR(36) PRIMARY KEY,
			name            VARCHAR(255) NOT NULL,
			description     TEXT,
			severity        VARCHAR(20) NOT NULL,
			enabled         TINYINT(1) DEFAULT 1,
			patterns        TEXT NOT NULL,
			match_locations TEXT NOT NULL,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS ip_list (
			id         VARCHAR(36) PRIMARY KEY,
			ip_address VARCHAR(45) NOT NULL,
			list_type  VARCHAR(20) NOT NULL,
			note       TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE INDEX idx_ip_list_unique (ip_address, list_type)
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			` + "`key`" + `   VARCHAR(255) PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sites (
			id                VARCHAR(36) PRIMARY KEY,
			name              VARCHAR(255) NOT NULL,
			domains           TEXT NOT NULL,
			upstream          VARCHAR(512) NOT NULL,
			enabled           TINYINT(1) DEFAULT 1,
			ai_enabled        TINYINT(1) DEFAULT 1,
			challenge_enabled TINYINT(1) DEFAULT 1,
			site_type         VARCHAR(50) DEFAULT 'website',
			created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
			INDEX idx_sites_enabled (enabled)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id        VARCHAR(36) PRIMARY KEY,
			timestamp DATETIME NOT NULL,
			actor     VARCHAR(100) NOT NULL,
			client_ip VARCHAR(45) NOT NULL,
			action    VARCHAR(50) NOT NULL,
			status    VARCHAR(20) NOT NULL,
			detail    TEXT,
			INDEX idx_audit_events_timestamp (timestamp),
			INDEX idx_audit_events_action (action),
			INDEX idx_audit_events_client_ip (client_ip)
		)`,
		`CREATE TABLE IF NOT EXISTS geo_rules (
			id         VARCHAR(36) PRIMARY KEY,
			country    VARCHAR(10) NOT NULL,
			action     VARCHAR(20) NOT NULL DEFAULT 'block',
			enabled    TINYINT(1) DEFAULT 1,
			created_at DATETIME NOT NULL,
			UNIQUE INDEX idx_geo_rules_country (country)
		)`,
		`CREATE TABLE IF NOT EXISTS users (
			id            VARCHAR(36) PRIMARY KEY,
			username      VARCHAR(100) NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			role          VARCHAR(20) NOT NULL DEFAULT 'viewer',
			created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.Exec(stmt); err != nil {
			return fmt.Errorf("create table: %w\nSQL: %s", err, stmt)
		}
	}
	return nil
}

// --- Settings ---

func (s *MySQLStore) GetSetting(key string) (string, error) {
	var value string
	err := s.db.QueryRow("SELECT value FROM settings WHERE `key` = ?", key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return value, err
}

func (s *MySQLStore) SetSetting(key, value string) error {
	_, err := s.db.Exec("INSERT INTO settings (`key`, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = VALUES(value)", key, value)
	return err
}

func (s *MySQLStore) ListSettings() (map[string]string, error) {
	rows, err := s.db.Query("SELECT `key`, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		m[k] = v
	}
	return m, nil
}

// --- Attack Logs ---

func (s *MySQLStore) InsertAttackLog(l model.AttackLog) error {
	headersJSON, _ := json.Marshal(l.Headers)
	_, err := s.db.Exec(
		`INSERT INTO attack_logs
		(id, timestamp, client_ip, site_id, site_name, domain, region, method, path, headers, body_preview, rule_id, rule_name, severity, source, action, ai_reasoning, reviewed, false_positive)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
		client_ip=VALUES(client_ip), region=VALUES(region), action=VALUES(action), ai_reasoning=VALUES(ai_reasoning)`,
		l.ID, l.Timestamp, l.ClientIP, l.SiteID, l.SiteName, l.Domain, l.Region, l.Method, l.Path,
		string(headersJSON), l.BodyPreview, l.RuleID, l.RuleName,
		l.Severity, l.Source, l.Action, l.AIReasoning, l.Reviewed, l.FalsePositive,
	)
	return err
}

func (s *MySQLStore) ListAttackLogs(offset, limit int, filter LogFilter) ([]model.AttackLog, int, error) {
	where := "1=1"
	args := []interface{}{}
	if filter.ClientIP != "" {
		where += " AND client_ip = ?"
		args = append(args, filter.ClientIP)
	}
	if filter.SiteID != "" {
		where += " AND site_id = ?"
		args = append(args, filter.SiteID)
	}
	if filter.Severity != "" {
		where += " AND severity = ?"
		args = append(args, filter.Severity)
	}
	if filter.Source != "" {
		where += " AND source = ?"
		args = append(args, filter.Source)
	}
	if !filter.Since.IsZero() {
		where += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := "SELECT id, timestamp, client_ip, site_id, site_name, domain, region, method, path, headers, body_preview, rule_id, rule_name, severity, source, action, ai_reasoning, reviewed, false_positive FROM attack_logs WHERE " + where + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []model.AttackLog
	for rows.Next() {
		var l model.AttackLog
		var headersJSON sql.NullString
		if err := rows.Scan(&l.ID, &l.Timestamp, &l.ClientIP, &l.SiteID, &l.SiteName, &l.Domain, &l.Region, &l.Method, &l.Path,
			&headersJSON, &l.BodyPreview, &l.RuleID, &l.RuleName,
			&l.Severity, &l.Source, &l.Action, &l.AIReasoning, &l.Reviewed, &l.FalsePositive); err != nil {
			return nil, 0, err
		}
		if headersJSON.Valid {
			l.Headers = headersJSON.String
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (s *MySQLStore) GetAttackLog(id string) (*model.AttackLog, error) {
	var l model.AttackLog
	var headersJSON sql.NullString
	err := s.db.QueryRow(
		"SELECT id, timestamp, client_ip, site_id, site_name, domain, region, method, path, headers, body_preview, rule_id, rule_name, severity, source, action, ai_reasoning, reviewed, false_positive FROM attack_logs WHERE id = ?", id,
	).Scan(&l.ID, &l.Timestamp, &l.ClientIP, &l.SiteID, &l.SiteName, &l.Domain, &l.Region, &l.Method, &l.Path,
		&headersJSON, &l.BodyPreview, &l.RuleID, &l.RuleName,
		&l.Severity, &l.Source, &l.Action, &l.AIReasoning, &l.Reviewed, &l.FalsePositive)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if headersJSON.Valid {
		l.Headers = headersJSON.String
	}
	return &l, nil
}

func (s *MySQLStore) GetAttackStats(since time.Time) (*AttackStats, error) {
	return s.GetAttackStatsBySite(since, "")
}

func (s *MySQLStore) GetAttackStatsBySite(since time.Time, siteID string) (*AttackStats, error) {
	stats := &AttackStats{
		BySeverity: make(map[string]int),
		BySource:   make(map[string]int),
	}
	where := "timestamp >= ?"
	args := []interface{}{since}
	if siteID != "" {
		where += " AND site_id = ?"
		args = append(args, siteID)
	}

	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where, args...).Scan(&stats.TotalRequests)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND action = 'blocked'", args...).Scan(&stats.BlockedCount)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai'", args...).Scan(&stats.AICount)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai' AND false_positive = 1", args...).Scan(&stats.AIFalsePositiveCount)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai' AND reviewed = 1", args...).Scan(&stats.AIReviewedCount)

	if rows, err := s.db.Query("SELECT severity, COUNT(*) FROM attack_logs WHERE "+where+" GROUP BY severity", args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var sev string
			var cnt int
			rows.Scan(&sev, &cnt)
			stats.BySeverity[sev] = cnt
		}
	}
	if rows, err := s.db.Query("SELECT source, COUNT(*) FROM attack_logs WHERE "+where+" GROUP BY source", args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var src string
			var cnt int
			rows.Scan(&src, &cnt)
			stats.BySource[src] = cnt
		}
	}
	if rows, err := s.db.Query("SELECT path, COUNT(*) as cnt FROM attack_logs WHERE "+where+" GROUP BY path ORDER BY cnt DESC LIMIT 10", args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var pc PathCount
			rows.Scan(&pc.Path, &pc.Count)
			stats.TopAttackPaths = append(stats.TopAttackPaths, pc)
		}
	}
	if rows, err := s.db.Query("SELECT region, COUNT(*) as cnt FROM attack_logs WHERE "+where+" AND region != '' GROUP BY region ORDER BY cnt DESC LIMIT 10", args...); err == nil {
		defer rows.Close()
		for rows.Next() {
			var rc RegionCount
			rows.Scan(&rc.Region, &rc.Count)
			stats.TopRegions = append(stats.TopRegions, rc)
		}
	}
	return stats, nil
}

func (s *MySQLStore) MarkAttackLogReview(id string, falsePositive bool) error {
	_, err := s.db.Exec("UPDATE attack_logs SET reviewed = 1, false_positive = ? WHERE id = ?", falsePositive, id)
	return err
}

func (s *MySQLStore) GetAIRuleSuggestions(since time.Time, minCount, limit int) ([]AIRuleSuggestion, error) {
	return s.GetAIRuleSuggestionsBySite(since, minCount, limit, "")
}

func (s *MySQLStore) GetAIRuleSuggestionsBySite(since time.Time, minCount, limit int, siteID string) ([]AIRuleSuggestion, error) {
	if minCount < 1 {
		minCount = 2
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	where := "timestamp >= ? AND source = 'ai' AND action = 'blocked' AND false_positive = 0"
	args := []interface{}{since}
	if siteID != "" {
		where += " AND site_id = ?"
		args = append(args, siteID)
	}
	args = append(args, minCount, limit)

	rows, err := s.db.Query(`
		SELECT path, rule_id, rule_name, severity, COUNT(*) AS cnt,
		       SUM(CASE WHEN reviewed = 1 THEN 1 ELSE 0 END) AS reviewed_cnt,
		       SUM(CASE WHEN false_positive = 1 THEN 1 ELSE 0 END) AS fp_cnt
		FROM attack_logs
		WHERE `+where+`
		GROUP BY path, rule_id, rule_name, severity
		HAVING cnt >= ?
		ORDER BY cnt DESC, path ASC
		LIMIT ?`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []AIRuleSuggestion
	for rows.Next() {
		var sgt AIRuleSuggestion
		if err := rows.Scan(&sgt.Path, &sgt.RuleID, &sgt.RuleName, &sgt.Severity, &sgt.Count, &sgt.Reviewed, &sgt.FalsePositive); err != nil {
			return nil, err
		}
		sgt.Key = sgt.RuleID + "|" + sgt.Path
		sgt.Pattern = "^" + regexpQuoteMeta(sgt.Path) + "$"
		out = append(out, sgt)
	}
	return out, nil
}

func (s *MySQLStore) CleanupOldLogs(retentionDays int) (int64, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	var total int64

	res, err := s.db.Exec("DELETE FROM attack_logs WHERE timestamp < ?", cutoff)
	if err != nil {
		return 0, fmt.Errorf("cleanup attack_logs: %w", err)
	}
	n, _ := res.RowsAffected()
	total += n

	res, err = s.db.Exec("DELETE FROM ssh_events WHERE timestamp < ?", cutoff)
	if err != nil {
		return total, fmt.Errorf("cleanup ssh_events: %w", err)
	}
	n, _ = res.RowsAffected()
	total += n

	if total > 0 {
		log.Printf("mysql log cleanup: removed %d records older than %d days", total, retentionDays)
	}
	return total, nil
}

// --- SSH Events ---

func (s *MySQLStore) InsertSSHEvent(e SSHEvent) error {
	_, err := s.db.Exec(
		`INSERT INTO ssh_events (id, timestamp, client_ip, region, username, event_type, message)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE region=VALUES(region), message=VALUES(message)`,
		e.ID, e.Timestamp, e.ClientIP, e.Region, e.Username, e.EventType, e.Message,
	)
	return err
}

func (s *MySQLStore) ListSSHEvents(offset, limit int, clientIP, eventType, username string) ([]SSHEvent, int, error) {
	where := "1=1"
	args := []interface{}{}
	if clientIP != "" {
		where += " AND client_ip = ?"
		args = append(args, clientIP)
	}
	if eventType != "" {
		where += " AND event_type = ?"
		args = append(args, eventType)
	}
	if username != "" {
		where += " AND username = ?"
		args = append(args, username)
	}

	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE "+where, args...).Scan(&total)

	args = append(args, limit, offset)
	rows, err := s.db.Query("SELECT id, timestamp, client_ip, region, username, event_type, message FROM ssh_events WHERE "+where+" ORDER BY timestamp DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []SSHEvent
	for rows.Next() {
		var e SSHEvent
		rows.Scan(&e.ID, &e.Timestamp, &e.ClientIP, &e.Region, &e.Username, &e.EventType, &e.Message)
		events = append(events, e)
	}
	return events, total, nil
}

func (s *MySQLStore) GetSSHStats(since time.Time) (map[string]interface{}, error) {
	var total, failed, blocked int
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ?", since).Scan(&total)
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ? AND event_type = 'failed'", since).Scan(&failed)
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ? AND event_type = 'blocked'", since).Scan(&blocked)

	type IPCount struct {
		IP     string `json:"ip"`
		Region string `json:"region"`
		Count  int    `json:"count"`
	}
	var topIPs []IPCount
	if rows, err := s.db.Query("SELECT client_ip, region, COUNT(*) as cnt FROM ssh_events WHERE timestamp >= ? AND event_type = 'failed' GROUP BY client_ip ORDER BY cnt DESC LIMIT 10", since); err == nil {
		defer rows.Close()
		for rows.Next() {
			var ic IPCount
			rows.Scan(&ic.IP, &ic.Region, &ic.Count)
			topIPs = append(topIPs, ic)
		}
	}

	return map[string]interface{}{
		"total":         total,
		"failed":        failed,
		"blocked":       blocked,
		"top_attackers": topIPs,
	}, nil
}

// --- Rules ---

func (s *MySQLStore) ListRules() ([]model.Rule, error) {
	rows, err := s.db.Query("SELECT id, name, description, severity, enabled, patterns, match_locations, created_at, updated_at FROM rules ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.Rule
	for rows.Next() {
		var r model.Rule
		var patternsJSON, locationsJSON string
		if err := rows.Scan(&r.ID, &r.Name, &r.Description, &r.Severity, &r.Enabled,
			&patternsJSON, &locationsJSON, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(patternsJSON), &r.Patterns)
		json.Unmarshal([]byte(locationsJSON), &r.MatchLocations)
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *MySQLStore) CreateRule(r model.Rule) error {
	patternsJSON, _ := json.Marshal(r.Patterns)
	locationsJSON, _ := json.Marshal(r.MatchLocations)
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT INTO rules (id, name, description, severity, enabled, patterns, match_locations, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)",
		r.ID, r.Name, r.Description, r.Severity, r.Enabled,
		string(patternsJSON), string(locationsJSON), now, now,
	)
	return err
}

func (s *MySQLStore) UpdateRule(r model.Rule) error {
	patternsJSON, _ := json.Marshal(r.Patterns)
	locationsJSON, _ := json.Marshal(r.MatchLocations)
	_, err := s.db.Exec(
		"UPDATE rules SET name=?, description=?, severity=?, enabled=?, patterns=?, match_locations=?, updated_at=? WHERE id=?",
		r.Name, r.Description, r.Severity, r.Enabled,
		string(patternsJSON), string(locationsJSON), time.Now(), r.ID,
	)
	return err
}

func (s *MySQLStore) DeleteRule(id string) error {
	_, err := s.db.Exec("DELETE FROM rules WHERE id = ?", id)
	return err
}

func (s *MySQLStore) GetRule(id string) (*model.Rule, error) {
	var r model.Rule
	var patternsJSON, locationsJSON string
	err := s.db.QueryRow(
		"SELECT id, name, description, severity, enabled, patterns, match_locations, created_at, updated_at FROM rules WHERE id = ?", id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Severity, &r.Enabled,
		&patternsJSON, &locationsJSON, &r.CreatedAt, &r.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(patternsJSON), &r.Patterns)
	json.Unmarshal([]byte(locationsJSON), &r.MatchLocations)
	return &r, nil
}

// --- IP List ---

func (s *MySQLStore) ListIPEntries(listType string) ([]model.IPEntry, error) {
	rows, err := s.db.Query("SELECT id, ip_address, list_type, note, created_at FROM ip_list WHERE list_type = ? ORDER BY created_at", listType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entries []model.IPEntry
	for rows.Next() {
		var e model.IPEntry
		if err := rows.Scan(&e.ID, &e.IPAddress, &e.ListType, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *MySQLStore) AddIPEntry(e model.IPEntry) error {
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT INTO ip_list (id, ip_address, list_type, note, created_at) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE note=VALUES(note), created_at=VALUES(created_at)",
		e.ID, e.IPAddress, e.ListType, e.Note, now,
	)
	return err
}

func (s *MySQLStore) RemoveIPEntry(id string) error {
	_, err := s.db.Exec("DELETE FROM ip_list WHERE id = ?", id)
	return err
}

func (s *MySQLStore) GetIPListMap(listType string) (map[string]bool, error) {
	entries, err := s.ListIPEntries(listType)
	if err != nil {
		return nil, err
	}
	m := make(map[string]bool, len(entries))
	for _, e := range entries {
		m[e.IPAddress] = true
	}
	return m, nil
}

func (s *MySQLStore) IsIPInList(ip, listType string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM ip_list WHERE ip_address = ? AND list_type = ?", ip, listType).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// --- Sites ---

func (s *MySQLStore) ListSites() ([]model.Site, error) {
	rows, err := s.db.Query("SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Site
	for rows.Next() {
		var site model.Site
		var domainsJSON string
		if err := rows.Scan(&site.ID, &site.Name, &domainsJSON, &site.Upstream, &site.Enabled, &site.AIEnabled, &site.ChallengeEnabled, &site.SiteType, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(domainsJSON), &site.Domains)
		out = append(out, site)
	}
	return out, nil
}

func (s *MySQLStore) ListEnabledSites() ([]model.Site, error) {
	rows, err := s.db.Query("SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites WHERE enabled = 1 ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Site
	for rows.Next() {
		var site model.Site
		var domainsJSON string
		if err := rows.Scan(&site.ID, &site.Name, &domainsJSON, &site.Upstream, &site.Enabled, &site.AIEnabled, &site.ChallengeEnabled, &site.SiteType, &site.CreatedAt, &site.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(domainsJSON), &site.Domains)
		out = append(out, site)
	}
	return out, nil
}

func (s *MySQLStore) GetSite(id string) (*model.Site, error) {
	var site model.Site
	var domainsJSON string
	err := s.db.QueryRow("SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites WHERE id = ?", id).
		Scan(&site.ID, &site.Name, &domainsJSON, &site.Upstream, &site.Enabled, &site.AIEnabled, &site.ChallengeEnabled, &site.SiteType, &site.CreatedAt, &site.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(domainsJSON), &site.Domains)
	return &site, nil
}

func (s *MySQLStore) CreateSite(site model.Site) error {
	domainsJSON, _ := json.Marshal(mysqlNormalizeDomains(site.Domains))
	now := time.Now()
	_, err := s.db.Exec("INSERT INTO sites(id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		site.ID, site.Name, string(domainsJSON), site.Upstream, site.Enabled, site.AIEnabled, site.ChallengeEnabled, site.SiteType, now, now)
	return err
}

func (s *MySQLStore) UpdateSite(site model.Site) error {
	domainsJSON, _ := json.Marshal(mysqlNormalizeDomains(site.Domains))
	_, err := s.db.Exec("UPDATE sites SET name=?, domains=?, upstream=?, enabled=?, ai_enabled=?, challenge_enabled=?, site_type=?, updated_at=? WHERE id=?",
		site.Name, string(domainsJSON), site.Upstream, site.Enabled, site.AIEnabled, site.ChallengeEnabled, site.SiteType, time.Now(), site.ID)
	return err
}

func (s *MySQLStore) DeleteSite(id string) error {
	_, err := s.db.Exec("DELETE FROM sites WHERE id = ?", id)
	return err
}

func mysqlNormalizeDomains(domains []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(domains))
	for _, d := range domains {
		d = strings.ToLower(strings.TrimSpace(d))
		d = strings.TrimPrefix(d, "http://")
		d = strings.TrimPrefix(d, "https://")
		d = strings.TrimSuffix(d, "/")
		if d != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	return out
}

// --- Geo ---

func (s *MySQLStore) InitGeoTable() error {
	return nil // already created in migrate()
}

func (s *MySQLStore) ListGeoRules() ([]model.GeoRule, error) {
	rows, err := s.db.Query("SELECT id, country, action, enabled, created_at FROM geo_rules ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []model.GeoRule
	for rows.Next() {
		var r model.GeoRule
		if err := rows.Scan(&r.ID, &r.Country, &r.Action, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *MySQLStore) AddGeoRule(r model.GeoRule) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	_, err := s.db.Exec(
		"INSERT INTO geo_rules (id, country, action, enabled, created_at) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE action=VALUES(action), enabled=VALUES(enabled)",
		r.ID, r.Country, r.Action, r.Enabled, time.Now(),
	)
	return err
}

func (s *MySQLStore) UpdateGeoRule(r model.GeoRule) error {
	existing, err := s.GetGeoRuleByID(r.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("geo rule not found: %s", r.ID)
	}
	if r.Country != "" {
		existing.Country = r.Country
	}
	if r.Action != "" {
		existing.Action = r.Action
	}
	existing.Enabled = r.Enabled
	_, err = s.db.Exec("UPDATE geo_rules SET country = ?, action = ?, enabled = ? WHERE id = ?",
		existing.Country, existing.Action, existing.Enabled, r.ID)
	return err
}

func (s *MySQLStore) GetGeoRuleByID(id string) (*model.GeoRule, error) {
	var r model.GeoRule
	err := s.db.QueryRow("SELECT id, country, action, enabled, created_at FROM geo_rules WHERE id = ?", id).
		Scan(&r.ID, &r.Country, &r.Action, &r.Enabled, &r.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func (s *MySQLStore) RemoveGeoRule(id string) error {
	_, err := s.db.Exec("DELETE FROM geo_rules WHERE id = ?", id)
	return err
}

func (s *MySQLStore) GetBlockedCountries() ([]string, error) {
	rows, err := s.db.Query("SELECT country FROM geo_rules WHERE action = 'block' AND enabled = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var countries []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		countries = append(countries, c)
	}
	return countries, nil
}

// --- Audit ---

func (s *MySQLStore) InsertAuditEvent(e model.AuditEvent) error {
	_, err := s.db.Exec(
		`INSERT INTO audit_events (id, timestamp, actor, client_ip, action, status, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE detail=VALUES(detail)`,
		e.ID, e.Timestamp, e.Actor, e.ClientIP, e.Action, e.Status, e.Detail,
	)
	return err
}

func (s *MySQLStore) ListAuditEvents(offset, limit int, filter AuditFilter) ([]model.AuditEvent, int, error) {
	where := "1=1"
	args := []interface{}{}
	if filter.Action != "" {
		where += " AND action = ?"
		args = append(args, filter.Action)
	}
	if filter.Status != "" {
		where += " AND status = ?"
		args = append(args, filter.Status)
	}
	if filter.Actor != "" {
		where += " AND actor = ?"
		args = append(args, filter.Actor)
	}
	if !filter.Since.IsZero() {
		where += " AND timestamp >= ?"
		args = append(args, filter.Since)
	}
	if !filter.Until.IsZero() {
		where += " AND timestamp <= ?"
		args = append(args, filter.Until)
	}

	var total int
	if err := s.db.QueryRow("SELECT COUNT(*) FROM audit_events WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	rows, err := s.db.Query("SELECT id, timestamp, actor, client_ip, action, status, detail FROM audit_events WHERE "+where+" ORDER BY timestamp DESC LIMIT ? OFFSET ?", args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []model.AuditEvent
	for rows.Next() {
		var e model.AuditEvent
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.Actor, &e.ClientIP, &e.Action, &e.Status, &e.Detail); err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}
	return events, total, nil
}

// --- Users ---

func (s *MySQLStore) ListUsers() ([]model.User, error) {
	rows, err := s.db.Query("SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *MySQLStore) GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	err := s.db.QueryRow("SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *MySQLStore) CreateUser(u model.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.Role == "" {
		u.Role = "viewer"
	}
	_, err := s.db.Exec(
		"INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.Username, u.PasswordHash, u.Role, time.Now(),
	)
	return err
}

func (s *MySQLStore) DeleteUser(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *MySQLStore) UpdateUserPassword(id string, hash string) error {
	_, err := s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	return err
}
