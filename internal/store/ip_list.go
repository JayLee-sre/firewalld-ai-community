package store

import (
	"time"

	"zhiyuwaf/internal/model"
)

func (s *Store) ListIPEntries(listType string) ([]model.IPEntry, error) {
	dbRows, err := s.db.Query("SELECT id, ip_address, list_type, note, created_at FROM ip_list WHERE list_type = ? ORDER BY created_at", listType)
	if err != nil {
		return nil, err
	}
	defer dbRows.Close()

	var entries []model.IPEntry
	for dbRows.Next() {
		var e model.IPEntry
		if err := dbRows.Scan(&e.ID, &e.IPAddress, &e.ListType, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (s *Store) AddIPEntry(e model.IPEntry) error {
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT OR REPLACE INTO ip_list (id, ip_address, list_type, note, created_at) VALUES (?, ?, ?, ?, ?)",
		e.ID, e.IPAddress, e.ListType, e.Note, now,
	)
	return err
}

func (s *Store) RemoveIPEntry(id string) error {
	_, err := s.db.Exec("DELETE FROM ip_list WHERE id = ?", id)
	return err
}

func (s *Store) GetIPListMap(listType string) (map[string]bool, error) {
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

func (s *Store) IsIPInList(ip, listType string) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM ip_list WHERE ip_address = ? AND list_type = ?", ip, listType).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
