package store

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/geo"
	"zhiyuwaf/internal/model"
)

func (s *Store) InitGeoTable() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS geo_rules (
			id           TEXT PRIMARY KEY,
			country      TEXT NOT NULL,
			country_code TEXT DEFAULT '',
			action       TEXT NOT NULL DEFAULT 'block',
			enabled      BOOLEAN DEFAULT 1,
			created_at   DATETIME NOT NULL
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_geo_rules_country ON geo_rules(country);
	`)
	if err != nil {
		return err
	}
	// Migrate: add country_code column for existing installs
	s.ensureColumns("geo_rules", map[string]string{
		"country_code": "ALTER TABLE geo_rules ADD COLUMN country_code TEXT DEFAULT ''",
	})
	// Backfill country_code for existing rules
	rows, _ := s.db.Query("SELECT id, country FROM geo_rules WHERE country_code = ''")
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id, name string
			if rows.Scan(&id, &name) == nil {
				if code := geo.ResolveCode(name); code != "" {
					s.db.Exec("UPDATE geo_rules SET country_code = ? WHERE id = ?", code, id)
				}
			}
		}
	}
	return nil
}

func (s *Store) ListGeoRules() ([]model.GeoRule, error) {
	rows, err := s.db.Query("SELECT id, country, COALESCE(country_code,''), action, enabled, created_at FROM geo_rules ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []model.GeoRule
	for rows.Next() {
		var r model.GeoRule
		if err := rows.Scan(&r.ID, &r.Country, &r.CountryCode, &r.Action, &r.Enabled, &r.CreatedAt); err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}
	return rules, nil
}

func (s *Store) AddGeoRule(r model.GeoRule) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	// Resolve country code from Chinese name if not provided
	if r.CountryCode == "" {
		r.CountryCode = geo.ResolveCode(r.Country)
	}
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO geo_rules (id, country, country_code, action, enabled, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		r.ID, r.Country, r.CountryCode, r.Action, r.Enabled, now,
	)
	return err
}

func (s *Store) UpdateGeoRule(r model.GeoRule) error {
	// Fetch existing rule to merge partial updates
	existing, err := s.GetGeoRuleByID(r.ID)
	if err != nil {
		return err
	}
	if existing == nil {
		return fmt.Errorf("geo rule not found: %s", r.ID)
	}
	if r.Country != "" {
		existing.Country = r.Country
		existing.CountryCode = geo.ResolveCode(r.Country)
	}
	if r.Action != "" {
		existing.Action = r.Action
	}
	existing.Enabled = r.Enabled
	_, err = s.db.Exec(
		"UPDATE geo_rules SET country = ?, country_code = ?, action = ?, enabled = ? WHERE id = ?",
		existing.Country, existing.CountryCode, existing.Action, existing.Enabled, r.ID,
	)
	return err
}

func (s *Store) GetGeoRuleByID(id string) (*model.GeoRule, error) {
	var r model.GeoRule
	err := s.db.QueryRow("SELECT id, country, COALESCE(country_code,''), action, enabled, created_at FROM geo_rules WHERE id = ?", id).
		Scan(&r.ID, &r.Country, &r.CountryCode, &r.Action, &r.Enabled, &r.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (s *Store) RemoveGeoRule(id string) error {
	_, err := s.db.Exec("DELETE FROM geo_rules WHERE id = ?", id)
	return err
}

func (s *Store) GetBlockedCountries() ([]string, error) {
	rows, err := s.db.Query("SELECT COALESCE(country_code,''), country FROM geo_rules WHERE action = 'block' AND enabled = 1")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code, name string
		if err := rows.Scan(&code, &name); err != nil {
			return nil, err
		}
		// Fallback: resolve code from name if migration hasn't backfilled
		if code == "" {
			code = geo.ResolveCode(name)
		}
		if code != "" {
			codes = append(codes, code)
		}
	}
	return codes, nil
}
