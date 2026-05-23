package model

import "time"

type ParsedRequest struct {
	ID          string
	Timestamp   time.Time
	ClientIP    string
	SiteID      string
	SiteName    string
	Domain      string
	SkipAI      bool
	Method      string
	URL         string
	Path        string
	QueryParams map[string][]string
	Headers     map[string][]string
	Body        []byte
	ContentType string
	UserAgent   string
	BodyPreview string
}

type DetectionResult struct {
	Blocked  bool
	RuleID   string
	RuleName string
	Severity string
	Message  string
	Source   string
	Latency  int64
}
