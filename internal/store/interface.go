package store

import (
	"time"

	"zhiyuwaf/internal/model"
)

// LogFilter holds query filters for attack log listing.
type LogFilter struct {
	ClientIP string
	SiteID   string
	Severity string
	Source   string
	Since    time.Time
}

// AuditFilter holds query filters for audit event listing.
type AuditFilter struct {
	Action string
	Status string
	Actor  string
	Since  time.Time
	Until  time.Time
}

// AttackStats summarizes attack log statistics.
type AttackStats struct {
	TotalRequests        int            `json:"total_requests"`
	BlockedCount         int            `json:"blocked_count"`
	AICount              int            `json:"ai_count"`
	AIFalsePositiveCount int            `json:"ai_false_positive_count"`
	AIReviewedCount      int            `json:"ai_reviewed_count"`
	BySeverity           map[string]int `json:"by_severity"`
	BySource             map[string]int `json:"by_source"`
	TopAttackPaths       []PathCount    `json:"top_attack_paths"`
	TopRegions           []RegionCount  `json:"top_regions"`
}

// PathCount represents a path with its attack count.
type PathCount struct {
	Path  string `json:"path"`
	Count int    `json:"count"`
}

// RegionCount represents a region with its attack count.
type RegionCount struct {
	Region string `json:"region"`
	Count  int    `json:"count"`
}

// AIRuleSuggestion represents an AI-suggested rule based on attack patterns.
type AIRuleSuggestion struct {
	Key           string `json:"key"`
	Path          string `json:"path"`
	RuleID        string `json:"rule_id"`
	RuleName      string `json:"rule_name"`
	Severity      string `json:"severity"`
	Count         int    `json:"count"`
	Pattern       string `json:"pattern"`
	Reviewed      int    `json:"reviewed"`
	FalsePositive int    `json:"false_positive"`
}

// SSHEvent represents an SSH monitoring event.
type SSHEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	ClientIP  string    `json:"client_ip"`
	Region    string    `json:"region"`
	Username  string    `json:"username"`
	EventType string    `json:"event_type"`
	Message   string    `json:"message"`
}

// Storage defines the interface for all storage backends.
type Storage interface {
	// Lifecycle
	Close() error

	// Settings
	GetSetting(key string) (string, error)
	SetSetting(key, value string) error
	ListSettings() (map[string]string, error)

	// Attack Logs
	InsertAttackLog(l model.AttackLog) error
	ListAttackLogs(offset, limit int, filter LogFilter) ([]model.AttackLog, int, error)
	GetAttackLog(id string) (*model.AttackLog, error)
	GetAttackStats(since time.Time) (*AttackStats, error)
	GetAttackStatsBySite(since time.Time, siteID string) (*AttackStats, error)
	MarkAttackLogReview(id string, falsePositive bool) error
	GetAIRuleSuggestions(since time.Time, minCount, limit int) ([]AIRuleSuggestion, error)
	GetAIRuleSuggestionsBySite(since time.Time, minCount, limit int, siteID string) ([]AIRuleSuggestion, error)
	CleanupOldLogs(retentionDays int) (int64, error)

	// SSH Events
	InsertSSHEvent(e SSHEvent) error
	ListSSHEvents(offset, limit int, clientIP, eventType, username string) ([]SSHEvent, int, error)
	GetSSHStats(since time.Time) (map[string]interface{}, error)

	// Rules
	ListRules() ([]model.Rule, error)
	CreateRule(r model.Rule) error
	UpdateRule(r model.Rule) error
	DeleteRule(id string) error
	GetRule(id string) (*model.Rule, error)

	// IP List
	ListIPEntries(listType string) ([]model.IPEntry, error)
	AddIPEntry(e model.IPEntry) error
	RemoveIPEntry(id string) error
	GetIPListMap(listType string) (map[string]bool, error)
	IsIPInList(ip, listType string) (bool, error)

	// Sites
	ListSites() ([]model.Site, error)
	ListEnabledSites() ([]model.Site, error)
	GetSite(id string) (*model.Site, error)
	CreateSite(site model.Site) error
	UpdateSite(site model.Site) error
	DeleteSite(id string) error

	// Geo
	InitGeoTable() error
	ListGeoRules() ([]model.GeoRule, error)
	AddGeoRule(r model.GeoRule) error
	UpdateGeoRule(r model.GeoRule) error
	GetGeoRuleByID(id string) (*model.GeoRule, error)
	RemoveGeoRule(id string) error
	GetBlockedCountries() ([]string, error)

	// Audit
	InsertAuditEvent(e model.AuditEvent) error
	ListAuditEvents(offset, limit int, filter AuditFilter) ([]model.AuditEvent, int, error)

	// Users
	ListUsers() ([]model.User, error)
	GetUserByUsername(username string) (*model.User, error)
	CreateUser(u model.User) error
	DeleteUser(id string) error
	UpdateUserPassword(id string, hash string) error
}
