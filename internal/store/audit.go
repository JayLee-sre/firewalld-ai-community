package store

import (
	"zhiyuwaf/internal/model"
)

// AuditFilter type is defined in interface.go.

func (s *Store) InsertAuditEvent(e model.AuditEvent) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO audit_events
		(id, timestamp, actor, client_ip, action, status, detail)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp, e.Actor, e.ClientIP, e.Action, e.Status, e.Detail,
	)
	return err
}

func (s *Store) ListAuditEvents(offset, limit int, filter AuditFilter) ([]model.AuditEvent, int, error) {
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

	query := "SELECT id, timestamp, actor, client_ip, action, status, detail FROM audit_events WHERE " + where + " ORDER BY timestamp DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
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
