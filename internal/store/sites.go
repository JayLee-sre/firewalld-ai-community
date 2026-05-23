package store

import (
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"zhiyuwaf/internal/model"
)

func (s *Store) ListSites() ([]model.Site, error) {
	rows, err := s.db.Query(`SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, nil
}

func (s *Store) ListEnabledSites() ([]model.Site, error) {
	rows, err := s.db.Query(`SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites WHERE enabled = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Site
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, site)
	}
	return out, nil
}

func (s *Store) GetSite(id string) (*model.Site, error) {
	row := s.db.QueryRow(`SELECT id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at FROM sites WHERE id = ?`, id)
	site, err := scanSite(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &site, nil
}

func (s *Store) CreateSite(site model.Site) error {
	domainsJSON, _ := json.Marshal(normalizeDomains(site.Domains))
	now := time.Now()
	_, err := s.db.Exec(`INSERT INTO sites(id, name, domains, upstream, enabled, ai_enabled, challenge_enabled, site_type, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		site.ID, site.Name, string(domainsJSON), site.Upstream, site.Enabled, site.AIEnabled, site.ChallengeEnabled, site.SiteType, now, now)
	return err
}

func (s *Store) UpdateSite(site model.Site) error {
	domainsJSON, _ := json.Marshal(normalizeDomains(site.Domains))
	_, err := s.db.Exec(`UPDATE sites SET name=?, domains=?, upstream=?, enabled=?, ai_enabled=?, challenge_enabled=?, site_type=?, updated_at=? WHERE id=?`,
		site.Name, string(domainsJSON), site.Upstream, site.Enabled, site.AIEnabled, site.ChallengeEnabled, site.SiteType, time.Now(), site.ID)
	return err
}

func (s *Store) DeleteSite(id string) error {
	_, err := s.db.Exec("DELETE FROM sites WHERE id = ?", id)
	return err
}

type siteScanner interface {
	Scan(dest ...interface{}) error
}

func scanSite(row siteScanner) (model.Site, error) {
	var site model.Site
	var domainsJSON string
	err := row.Scan(&site.ID, &site.Name, &domainsJSON, &site.Upstream, &site.Enabled, &site.AIEnabled, &site.ChallengeEnabled, &site.SiteType, &site.CreatedAt, &site.UpdatedAt)
	if err != nil {
		return site, err
	}
	_ = json.Unmarshal([]byte(domainsJSON), &site.Domains)
	return site, nil
}

func normalizeDomains(domains []string) []string {
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
