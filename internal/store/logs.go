package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"zhiyuwaf/internal/model"
)

// Type definitions (LogFilter, AttackStats, etc.) are in interface.go.

func (s *Store) InsertAttackLog(l model.AttackLog) error {
	headersJSON, _ := json.Marshal(l.Headers)
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO attack_logs
		(id, timestamp, client_ip, site_id, site_name, domain, region, method, path, headers, body_preview, rule_id, rule_name, severity, source, action, ai_reasoning, reviewed, false_positive)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.ID, l.Timestamp, l.ClientIP, l.SiteID, l.SiteName, l.Domain, l.Region, l.Method, l.Path,
		string(headersJSON), l.BodyPreview, l.RuleID, l.RuleName,
		l.Severity, l.Source, l.Action, l.AIReasoning, l.Reviewed, l.FalsePositive,
	)
	return err
}

func (s *Store) ListAttackLogs(offset, limit int, filter LogFilter) ([]model.AttackLog, int, error) {
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
	err := s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where, args...).Scan(&total)
	if err != nil {
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

func (s *Store) GetAttackLog(id string) (*model.AttackLog, error) {
	var l model.AttackLog
	var headersJSON sql.NullString
	err := s.db.QueryRow(
		"SELECT id, timestamp, client_ip, site_id, site_name, domain, region, method, path, headers, body_preview, rule_id, rule_name, severity, source, action, ai_reasoning, reviewed, false_positive FROM attack_logs WHERE id = ?",
		id,
	).Scan(&l.ID, &l.Timestamp, &l.ClientIP, &l.SiteID, &l.SiteName, &l.Domain, &l.Region, &l.Method, &l.Path,
		&headersJSON, &l.BodyPreview, &l.RuleID, &l.RuleName,
		&l.Severity, &l.Source, &l.Action, &l.AIReasoning, &l.Reviewed, &l.FalsePositive)
	if err == sql.ErrNoRows {
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

func (s *Store) GetAttackStats(since time.Time) (*AttackStats, error) {
	return s.GetAttackStatsBySite(since, "")
}

func (s *Store) GetAttackStatsBySite(since time.Time, siteID string) (*AttackStats, error) {
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

	err := s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where, args...).Scan(&stats.TotalRequests)
	if err != nil {
		return nil, err
	}

	blockedArgs := append([]interface{}{}, args...)
	err = s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND action = 'blocked'", blockedArgs...).Scan(&stats.BlockedCount)
	if err != nil {
		return nil, err
	}

	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai'", args...).Scan(&stats.AICount)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai' AND false_positive = 1", args...).Scan(&stats.AIFalsePositiveCount)
	s.db.QueryRow("SELECT COUNT(*) FROM attack_logs WHERE "+where+" AND source = 'ai' AND reviewed = 1", args...).Scan(&stats.AIReviewedCount)

	rows, err := s.db.Query("SELECT severity, COUNT(*) FROM attack_logs WHERE "+where+" GROUP BY severity", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var sev string
		var cnt int
		rows.Scan(&sev, &cnt)
		stats.BySeverity[sev] = cnt
	}

	rows2, err := s.db.Query("SELECT source, COUNT(*) FROM attack_logs WHERE "+where+" GROUP BY source", args...)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var src string
		var cnt int
		rows2.Scan(&src, &cnt)
		stats.BySource[src] = cnt
	}

	rows3, err := s.db.Query("SELECT path, COUNT(*) as cnt FROM attack_logs WHERE "+where+" GROUP BY path ORDER BY cnt DESC LIMIT 10", args...)
	if err != nil {
		return nil, err
	}
	defer rows3.Close()
	for rows3.Next() {
		var pc PathCount
		rows3.Scan(&pc.Path, &pc.Count)
		stats.TopAttackPaths = append(stats.TopAttackPaths, pc)
	}

	rows4, err := s.db.Query("SELECT region, COUNT(*) as cnt FROM attack_logs WHERE "+where+" AND region != '' GROUP BY region ORDER BY cnt DESC LIMIT 10", args...)
	if err == nil {
		defer rows4.Close()
		for rows4.Next() {
			var rc RegionCount
			rows4.Scan(&rc.Region, &rc.Count)
			stats.TopRegions = append(stats.TopRegions, rc)
		}
	}

	return stats, nil
}

func (s *Store) MarkAttackLogReview(id string, falsePositive bool) error {
	_, err := s.db.Exec("UPDATE attack_logs SET reviewed = 1, false_positive = ? WHERE id = ?", falsePositive, id)
	return err
}

func (s *Store) GetAIRuleSuggestions(since time.Time, minCount, limit int) ([]AIRuleSuggestion, error) {
	return s.GetAIRuleSuggestionsBySite(since, minCount, limit, "")
}

func (s *Store) GetAIRuleSuggestionsBySite(since time.Time, minCount, limit int, siteID string) ([]AIRuleSuggestion, error) {
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

func regexpQuoteMeta(s string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`.`, `\.`,
		`+`, `\+`,
		`*`, `\*`,
		`?`, `\?`,
		`(`, `\(`,
		`)`, `\)`,
		`[`, `\[`,
		`]`, `\]`,
		`{`, `\{`,
		`}`, `\}`,
		`^`, `\^`,
		`$`, `\$`,
		`|`, `\|`,
	)
	return replacer.Replace(s)
}

// SSHEvent type is defined in interface.go.

func (s *Store) InsertSSHEvent(e SSHEvent) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO ssh_events (id, timestamp, client_ip, region, username, event_type, message) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.Timestamp, e.ClientIP, e.Region, e.Username, e.EventType, e.Message,
	)
	return err
}

func (s *Store) ListSSHEvents(offset, limit int, clientIP, eventType, username string) ([]SSHEvent, int, error) {
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

func (s *Store) GetSSHStats(since time.Time) (map[string]interface{}, error) {
	var total int
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ?", since).Scan(&total)

	var failed int
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ? AND event_type = 'failed'", since).Scan(&failed)

	var blocked int
	s.db.QueryRow("SELECT COUNT(*) FROM ssh_events WHERE timestamp >= ? AND event_type = 'blocked'", since).Scan(&blocked)

	type IPCount struct {
		IP     string `json:"ip"`
		Region string `json:"region"`
		Count  int    `json:"count"`
	}
	var topIPs []IPCount

	rows, err := s.db.Query("SELECT client_ip, region, COUNT(*) as cnt FROM ssh_events WHERE timestamp >= ? AND event_type = 'failed' GROUP BY client_ip ORDER BY cnt DESC LIMIT 10", since)
	if err != nil {
		log.Printf("GetSSHStats top attackers query failed: %v", err)
	} else {
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

// CleanupOldLogs deletes attack logs and SSH events older than retentionDays.
// Returns the number of rows deleted.
func (s *Store) CleanupOldLogs(retentionDays int) (int64, error) {
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
		log.Printf("log cleanup: removed %d records older than %d days", total, retentionDays)
	}
	return total, nil
}
