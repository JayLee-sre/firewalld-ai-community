package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS attack_logs (
		id           TEXT PRIMARY KEY,
		timestamp    DATETIME NOT NULL,
		client_ip    TEXT NOT NULL,
		region       TEXT DEFAULT '',
		method       TEXT NOT NULL,
		path         TEXT NOT NULL,
		headers      TEXT,
		body_preview TEXT,
		rule_id      TEXT,
		rule_name    TEXT,
		severity     TEXT,
		source       TEXT,
		action       TEXT DEFAULT 'blocked',
		ai_reasoning TEXT,
		reviewed     BOOLEAN DEFAULT 0,
		false_positive BOOLEAN DEFAULT 0
	);
	CREATE INDEX IF NOT EXISTS idx_attack_logs_timestamp ON attack_logs(timestamp);
	CREATE INDEX IF NOT EXISTS idx_attack_logs_client_ip ON attack_logs(client_ip);
	CREATE INDEX IF NOT EXISTS idx_attack_logs_severity  ON attack_logs(severity);

	CREATE TABLE IF NOT EXISTS ssh_events (
		id         TEXT PRIMARY KEY,
		timestamp  DATETIME NOT NULL,
		client_ip  TEXT NOT NULL,
		region     TEXT DEFAULT '',
		username   TEXT,
		event_type TEXT NOT NULL,
		message    TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_ssh_events_timestamp ON ssh_events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_ssh_events_client_ip ON ssh_events(client_ip);

	CREATE TABLE IF NOT EXISTS rules (
		id              TEXT PRIMARY KEY,
		name            TEXT NOT NULL,
		description     TEXT,
		severity        TEXT NOT NULL,
		enabled         BOOLEAN DEFAULT 1,
		patterns        TEXT NOT NULL,
		match_locations TEXT NOT NULL,
		created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS ip_list (
		id         TEXT PRIMARY KEY,
		ip_address TEXT NOT NULL,
		list_type  TEXT NOT NULL,
		note       TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_ip_list_unique ON ip_list(ip_address, list_type);

	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sites (
		id                TEXT PRIMARY KEY,
		name              TEXT NOT NULL,
		domains           TEXT NOT NULL,
		upstream          TEXT NOT NULL,
		enabled           BOOLEAN DEFAULT 1,
		ai_enabled        BOOLEAN DEFAULT 1,
		challenge_enabled BOOLEAN DEFAULT 1,
		site_type         TEXT DEFAULT 'website',
		created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_sites_enabled ON sites(enabled);

	CREATE TABLE IF NOT EXISTS audit_events (
		id         TEXT PRIMARY KEY,
		timestamp  DATETIME NOT NULL,
		actor      TEXT NOT NULL,
		client_ip  TEXT NOT NULL,
		action     TEXT NOT NULL,
		status     TEXT NOT NULL,
		detail     TEXT
	);
	CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp);
	CREATE INDEX IF NOT EXISTS idx_audit_events_action ON audit_events(action);
	CREATE INDEX IF NOT EXISTS idx_audit_events_client_ip ON audit_events(client_ip);

	CREATE TABLE IF NOT EXISTS users (
		id            TEXT PRIMARY KEY,
		username      TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role          TEXT NOT NULL DEFAULT 'viewer',
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.ensureColumns("attack_logs", map[string]string{
		"reviewed":       "ALTER TABLE attack_logs ADD COLUMN reviewed BOOLEAN DEFAULT 0",
		"false_positive": "ALTER TABLE attack_logs ADD COLUMN false_positive BOOLEAN DEFAULT 0",
		"site_id":        "ALTER TABLE attack_logs ADD COLUMN site_id TEXT DEFAULT ''",
		"site_name":      "ALTER TABLE attack_logs ADD COLUMN site_name TEXT DEFAULT ''",
		"domain":         "ALTER TABLE attack_logs ADD COLUMN domain TEXT DEFAULT ''",
	}); err != nil {
		return err
	}

	// Migrate existing admin password to users table
	var userCount int
	s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if userCount == 0 {
		if hash, _ := s.GetSetting("admin_password_hash"); hash != "" {
			s.db.Exec("INSERT INTO users (id, username, password_hash, role) VALUES (?, 'admin', ?, 'admin')",
				"admin-default", hash)
		}
	}
	return nil
}

func (s *Store) ensureColumns(table string, alters map[string]string) error {
	rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	exists := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue interface{}
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		exists[name] = true
	}
	for column, stmt := range alters {
		if !exists[column] {
			if _, err := s.db.Exec(stmt); err != nil {
				return err
			}
		}
	}
	return nil
}
