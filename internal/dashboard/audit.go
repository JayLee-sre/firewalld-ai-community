package dashboard

import (
	"log"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

func (s *Server) recordAudit(actor, clientIP, action, status, detail string) {
	if actor == "" {
		actor = "system"
	}
	if clientIP == "" {
		clientIP = "-"
	}
	event := model.AuditEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		Actor:     actor,
		ClientIP:  clientIP,
		Action:    action,
		Status:    status,
		Detail:    detail,
	}
	if err := s.store.InsertAuditEvent(event); err != nil {
		log.Printf("failed to save audit event: %v", err)
	}
}
