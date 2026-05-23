package model

import "time"

type AuditEvent struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	ClientIP  string    `json:"client_ip"`
	Action    string    `json:"action"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail"`
}
