package store

import (
	"database/sql"
	"encoding/json"
	"time"

	"zhiyuwaf/internal/model"
)

func (s *Store) ListRules() ([]model.Rule, error) {
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

func (s *Store) CreateRule(r model.Rule) error {
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

func (s *Store) UpdateRule(r model.Rule) error {
	patternsJSON, _ := json.Marshal(r.Patterns)
	locationsJSON, _ := json.Marshal(r.MatchLocations)
	_, err := s.db.Exec(
		"UPDATE rules SET name=?, description=?, severity=?, enabled=?, patterns=?, match_locations=?, updated_at=? WHERE id=?",
		r.Name, r.Description, r.Severity, r.Enabled,
		string(patternsJSON), string(locationsJSON), time.Now(), r.ID,
	)
	return err
}

func (s *Store) DeleteRule(id string) error {
	_, err := s.db.Exec("DELETE FROM rules WHERE id = ?", id)
	return err
}

func (s *Store) GetRule(id string) (*model.Rule, error) {
	var r model.Rule
	var patternsJSON, locationsJSON string
	err := s.db.QueryRow(
		"SELECT id, name, description, severity, enabled, patterns, match_locations, created_at, updated_at FROM rules WHERE id = ?", id,
	).Scan(&r.ID, &r.Name, &r.Description, &r.Severity, &r.Enabled,
		&patternsJSON, &locationsJSON, &r.CreatedAt, &r.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(patternsJSON), &r.Patterns)
	json.Unmarshal([]byte(locationsJSON), &r.MatchLocations)
	return &r, nil
}
