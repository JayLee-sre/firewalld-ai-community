package dashboard

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"

	"zhiyuwaf/internal/license"
)

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := s.store.ListSettings()
	if err != nil {
		http.Error(w, `{"error":"failed to get settings"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, settings)
}

// settingsProtectedKeys are keys that cannot be modified via the general settings API.
// These can only be set through their dedicated endpoints (license activation, etc).
var settingsProtectedKeys = map[string]bool{
	"license_token":       true,
	"license_key":         true,
	"license_edition":     true,
	"license_customer":    true,
	"license_expires_at":  true,
	"license_machine_id":  true,
	"license_next_check_at": true,
	"license_grace_until": true,
	"admin_password_hash": true,
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var settings map[string]string
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	for k, v := range settings {
		if settingsProtectedKeys[k] {
			continue // silently skip protected keys
		}
		if err := s.store.SetSetting(k, v); err != nil {
			http.Error(w, `{"error":"failed to save settings"}`, http.StatusInternalServerError)
			return
		}
	}
	s.recordAudit("admin", dashboardClientIP(r), "settings_update", "success", "settings updated")

	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleReloadConfig(w http.ResponseWriter, r *http.Request) {
	if s.OnConfigReload != nil {
		s.OnConfigReload()
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "reloaded",
		"message": "配置已重新加载",
	})
}

func (s *Server) handleActivateLicense(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var body struct {
		LicenseKey string `json:"license_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if body.LicenseKey == "" {
		http.Error(w, `{"error":"license_key is required"}`, http.StatusBadRequest)
		return
	}

	client, err := license.NewClient(s.cfg.License.CenterURL, s.cfg.License.PublicKey, time.Duration(s.cfg.License.Timeout)*time.Second)
	if err != nil {
		s.recordAudit("admin", dashboardClientIP(r), "license_activate", "failed", "license client configuration invalid")
		http.Error(w, `{"error":"license client configuration invalid"}`, http.StatusInternalServerError)
		return
	}
	machineID := license.MachineID()
	hostname, _ := os.Hostname()
	resp, err := client.Activate(r.Context(), license.ActivationRequest{
		LicenseKey: strings.TrimSpace(body.LicenseKey),
		MachineID:  machineID,
		Hostname:   hostname,
		Version:    "ZhiYu-WAF v1.0.0",
	})
	if err != nil {
		s.recordAudit("admin", dashboardClientIP(r), "license_activate", "failed", "license center activation failed: "+err.Error())
		http.Error(w, `{"error":"授权中心校验失败，请确认授权码、机器数量和网络连通性"}`, http.StatusBadRequest)
		return
	}
	if err := license.IsUsable(resp.License, machineID, time.Now()); err != nil {
		s.recordAudit("admin", dashboardClientIP(r), "license_activate", "failed", "license token unusable: "+err.Error())
		http.Error(w, `{"error":"license token unusable"}`, http.StatusBadRequest)
		return
	}

	if err := s.store.SetSetting("license_key", resp.LicenseKey); err != nil {
		http.Error(w, `{"error":"failed to save license"}`, http.StatusInternalServerError)
		return
	}
	_ = s.store.SetSetting("license_token", resp.Token)
	_ = s.store.SetSetting("license_edition", resp.License.Edition)
	_ = s.store.SetSetting("license_customer", resp.License.Customer)
	_ = s.store.SetSetting("license_expires_at", resp.License.ExpiresAt)
	_ = s.store.SetSetting("license_machine_id", resp.License.MachineID)
	_ = s.store.SetSetting("license_next_check_at", time.Unix(resp.License.NextCheckAt, 0).Format(time.RFC3339))
	_ = s.store.SetSetting("license_grace_until", time.Unix(resp.License.GraceUntil, 0).Format(time.RFC3339))
	s.recordAudit("admin", dashboardClientIP(r), "license_activate", "success", "professional edition activated via license center")

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        resp.Status,
		"edition":       resp.License.Edition,
		"customer":      resp.License.Customer,
		"expires_at":    resp.License.ExpiresAt,
		"machine_id":    resp.License.MachineID,
		"features":      resp.License.Features,
		"next_check_at": resp.License.NextCheckAt,
		"message":       "授权已通过授权中心激活，专业版功能已启用",
	})
}
