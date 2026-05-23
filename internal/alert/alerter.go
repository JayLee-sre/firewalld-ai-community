package alert

import "time"

// Alert represents a WAF alert to be sent via notification channels.
type Alert struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Severity  string    `json:"severity"`
	Message   string    `json:"message"`
	SourceIP  string    `json:"source_ip,omitempty"`
	RuleID    string    `json:"rule_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// Alerter is the interface for alert notification channels.
type Alerter interface {
	Send(alert Alert) error
	Name() string
}
