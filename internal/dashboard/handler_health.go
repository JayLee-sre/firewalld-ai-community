package dashboard

import (
	"context"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"zhiyuwaf/internal/license"
)

var startTime = time.Now()

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	s.refreshLicenseIfNeeded(true)
	edition := s.currentEdition()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"product":      "ZhiYu-WAF v1.0.0",
		"edition":      edition,
		"license":      s.currentLicenseInfo(),
		"license_mode": s.cfg.License.CenterURL,
		"status":       "ok",
		"hostname":     hostname,
		"uptime":       time.Since(startTime).String(),
	})
}

func (s *Server) handleHealthDetail(w http.ResponseWriter, r *http.Request) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	hostname, _ := os.Hostname()
	s.refreshLicenseIfNeeded(true)
	edition := s.currentEdition()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"product":       "ZhiYu-WAF v1.0.0",
		"edition":       edition,
		"license":       s.currentLicenseInfo(),
		"license_mode":  s.cfg.License.CenterURL,
		"status":        "ok",
		"hostname":      hostname,
		"uptime":        time.Since(startTime).String(),
		"go_version":    runtime.Version(),
		"goroutines":    runtime.NumGoroutine(),
		"memory_mb":     mem.Alloc / 1024 / 1024,
		"memory_sys_mb": mem.Sys / 1024 / 1024,
		"gc_runs":       mem.NumGC,
		"ai_enabled":    s.cfg.AI.Enabled,
	})
}

func (s *Server) currentEdition() string {
	token, _ := s.store.GetSetting("license_token")
	if token == "" {
		return "community"
	}
	client, err := license.NewClient(s.cfg.License.CenterURL, s.cfg.License.PublicKey, time.Duration(s.cfg.License.Timeout)*time.Second)
	if err != nil {
		return "community"
	}
	payload, err := license.VerifyToken(token, client.PublicKey)
	if err != nil {
		return "community"
	}
	if err := license.IsUsable(payload, license.MachineID(), time.Now()); err != nil {
		return "community"
	}
	if payload.Edition == "pro" {
		return "pro"
	}
	return "community"
}

func (s *Server) currentLicenseInfo() map[string]interface{} {
	info := map[string]interface{}{
		"status":     "community",
		"machine_id": license.MachineID(),
	}
	token, _ := s.store.GetSetting("license_token")
	if token == "" {
		return info
	}
	client, err := license.NewClient(s.cfg.License.CenterURL, s.cfg.License.PublicKey, time.Duration(s.cfg.License.Timeout)*time.Second)
	if err != nil {
		info["status"] = "invalid_config"
		return info
	}
	payload, err := license.VerifyToken(token, client.PublicKey)
	if err != nil {
		info["status"] = "invalid_signature"
		return info
	}
	if err := license.IsUsable(payload, license.MachineID(), time.Now()); err != nil {
		info["status"] = "invalid"
		info["reason"] = err.Error()
		return info
	}
	info["status"] = "active"
	info["edition"] = payload.Edition
	info["customer"] = payload.Customer
	info["expires_at"] = payload.ExpiresAt
	info["features"] = payload.Features
	info["next_check_at"] = time.Unix(payload.NextCheckAt, 0).Format(time.RFC3339)
	info["grace_until"] = time.Unix(payload.GraceUntil, 0).Format(time.RFC3339)
	return info
}

func (s *Server) refreshLicenseIfNeeded(force bool) {
	token, _ := s.store.GetSetting("license_token")
	licenseKey, _ := s.store.GetSetting("license_key")
	if token == "" || licenseKey == "" {
		return
	}
	client, err := license.NewClient(s.cfg.License.CenterURL, s.cfg.License.PublicKey, time.Duration(s.cfg.License.Timeout)*time.Second)
	if err != nil {
		return
	}
	payload, err := license.VerifyToken(token, client.PublicKey)
	if err != nil {
		return
	}
	if !force && (payload.NextCheckAt == 0 || time.Now().Unix() < payload.NextCheckAt) {
		return
	}
	hostname, _ := os.Hostname()
	resp, err := client.VerifyOnline(context.Background(), license.ActivationRequest{
		LicenseKey: licenseKey,
		MachineID:  license.MachineID(),
		Hostname:   hostname,
		Version:    "ZhiYu-WAF v1.0.0",
	})
	if err != nil {
		errMsg := err.Error()
		// License explicitly revoked — clear immediately
		if strings.Contains(errMsg, "status=401") || strings.Contains(errMsg, "status=403") {
			s.clearLocalLicense("license center rejected current license: " + errMsg)
			return
		}
		// Network error — enforce grace period from token, don't silently keep Pro forever
		if payload.GraceUntil > 0 && time.Now().Unix() > payload.GraceUntil {
			s.clearLocalLicense("license grace period expired after connectivity loss")
		}
		return
	}
	_ = s.store.SetSetting("license_token", resp.Token)
	_ = s.store.SetSetting("license_edition", resp.License.Edition)
	_ = s.store.SetSetting("license_customer", resp.License.Customer)
	_ = s.store.SetSetting("license_expires_at", resp.License.ExpiresAt)
	_ = s.store.SetSetting("license_machine_id", resp.License.MachineID)
	_ = s.store.SetSetting("license_next_check_at", time.Unix(resp.License.NextCheckAt, 0).Format(time.RFC3339))
	_ = s.store.SetSetting("license_grace_until", time.Unix(resp.License.GraceUntil, 0).Format(time.RFC3339))
}

func (s *Server) clearLocalLicense(reason string) {
	_ = s.store.SetSetting("license_token", "")
	_ = s.store.SetSetting("license_edition", "")
	_ = s.store.SetSetting("license_customer", "")
	_ = s.store.SetSetting("license_expires_at", "")
	_ = s.store.SetSetting("license_machine_id", license.MachineID())
	_ = s.store.SetSetting("license_next_check_at", "")
	_ = s.store.SetSetting("license_grace_until", "")
	s.recordAudit("system", "", "license_verify", "revoked", reason)
}
