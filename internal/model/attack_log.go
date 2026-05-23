package model

import "time"

type AttackLog struct {
	ID            string    `json:"id"`
	Timestamp     time.Time `json:"timestamp"`
	ClientIP      string    `json:"client_ip"`
	SiteID        string    `json:"site_id"`
	SiteName      string    `json:"site_name"`
	Domain        string    `json:"domain"`
	Region        string    `json:"region"`
	Method        string    `json:"method"`
	Path          string    `json:"path"`
	Headers       string    `json:"headers"`
	BodyPreview   string    `json:"body_preview"`
	RuleID        string    `json:"rule_id"`
	RuleName      string    `json:"rule_name"`
	Severity      string    `json:"severity"`
	Source        string    `json:"source"`
	Action        string    `json:"action"`
	AIReasoning   string    `json:"ai_reasoning"`
	Reviewed      bool      `json:"reviewed"`
	FalsePositive bool      `json:"false_positive"`
}
