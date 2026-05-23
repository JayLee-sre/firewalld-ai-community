package dashboard

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	storedHash, _ := s.store.GetSetting("admin_password_hash")
	// If password has been changed from the initial OTP (bcrypt hash doesn't start with OTP pattern),
	// setup is not needed. We check by seeing if "setup_done" setting exists.
	setupDone, _ := s.store.GetSetting("setup_done")

	needed := setupDone != "true" && storedHash != ""
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"needed": needed,
	})
}

func (s *Server) handleSetupPassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Password) < 12 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "密码至少12个字符"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "密码加密失败"})
		return
	}
	if err := s.store.SetSetting("admin_password_hash", string(hash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "保存密码失败"})
		return
	}

	log.Println("setup: admin password updated")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleSetupApply(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req struct {
		// Password (optional, may have been set in step 1)
		Password string `json:"password"`
		// Proxy
		BackendAddr    string `json:"backend_addr"`
		ListenPort     int    `json:"listen_port"`
		IPTablesEnable bool   `json:"iptables_enable"`
		IPTablesPort   int    `json:"iptables_port"`
		// AI
		AIEnabled  bool   `json:"ai_enabled"`
		APIKey     string `json:"api_key"`
		AIModel    string `json:"ai_model"`
		AIBaseURL  string `json:"ai_base_url"`
		// Security
		RPM          int  `json:"rpm"`
		BurstSize    int  `json:"burst_size"`
		SSHEnabled   bool `json:"ssh_enabled"`
		SSHMaxFails  int  `json:"ssh_max_fails"`
		SSHBanMins   int  `json:"ssh_ban_minutes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "请求格式错误"})
		return
	}

	if strings.TrimSpace(req.BackendAddr) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "后端地址不能为空"})
		return
	}

	// Update password if provided
	if req.Password != "" && len(req.Password) >= 12 {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err == nil {
			s.store.SetSetting("admin_password_hash", string(hash))
		}
	}

	// Apply proxy config
	s.cfg.Proxy.BackendAddr = req.BackendAddr
	if req.ListenPort > 0 && req.ListenPort <= 65535 {
		s.cfg.Proxy.ListenAddr = fmt.Sprintf(":%d", req.ListenPort)
	}
	s.cfg.Proxy.IPTablesEnable = req.IPTablesEnable
	if req.IPTablesPort > 0 {
		s.cfg.Proxy.IPTablesPort = req.IPTablesPort
	}

	// Apply AI config
	s.cfg.AI.Enabled = req.AIEnabled
	s.cfg.AI.Provider = "openai"
	if req.APIKey != "" {
		s.cfg.AI.Providers.OpenAI.APIKey = req.APIKey
	}
	if req.AIModel != "" {
		s.cfg.AI.Providers.OpenAI.Model = req.AIModel
	}
	if req.AIBaseURL != "" {
		s.cfg.AI.Providers.OpenAI.BaseURL = req.AIBaseURL
	}

	// Apply engine config
	if req.RPM > 0 {
		s.cfg.Engine.RateLimit.RequestsPerMinute = req.RPM
	}
	if req.BurstSize > 0 {
		s.cfg.Engine.RateLimit.BurstSize = req.BurstSize
	}

	// Apply SSH config
	s.cfg.SSH.Enabled = req.SSHEnabled
	if req.SSHMaxFails > 0 {
		s.cfg.SSH.MaxFails = req.SSHMaxFails
	}
	if req.SSHBanMins > 0 {
		s.cfg.SSH.BanMinutes = req.SSHBanMins
	}

	// Persist AI settings to DB (same as existing handlers)
	s.store.SetSetting("ai_enabled", fmt.Sprintf("%v", s.cfg.AI.Enabled))
	s.store.SetSetting("ai_provider", s.cfg.AI.Provider)
	if s.cfg.AI.Providers.OpenAI.APIKey != "" {
		s.store.SetSetting("ai_openai_api_key", s.cfg.AI.Providers.OpenAI.APIKey)
	}
	if s.cfg.AI.Providers.OpenAI.Model != "" {
		s.store.SetSetting("ai_openai_model", s.cfg.AI.Providers.OpenAI.Model)
	}
	if s.cfg.AI.Providers.OpenAI.BaseURL != "" {
		s.store.SetSetting("ai_openai_base_url", s.cfg.AI.Providers.OpenAI.BaseURL)
	}

	// Write config YAML
	if s.configPath != "" {
		if err := s.writeConfigYAML(); err != nil {
			log.Printf("setup: failed to write config: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "配置写入失败: " + err.Error()})
			return
		}
	}

	// Mark setup as done
	s.store.SetSetting("setup_done", "true")

	// Trigger config reload + AI reinit
	if s.OnConfigReload != nil {
		s.OnConfigReload()
	}
	if s.OnAIConfigChanged != nil {
		s.OnAIConfigChanged()
	}

	s.recordAudit("admin", dashboardClientIP(r), "setup_completed", "success", "")
	log.Println("setup: configuration applied successfully")

	// Auto-login: generate JWT so user doesn't need to re-enter password
	token, err := GenerateToken(s.cfg.Dashboard.JWTSecret, "admin", "admin")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"status": "ok", "token": token})
}

func (s *Server) writeConfigYAML() error {
	data, err := yaml.Marshal(s.cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(s.configPath, data, 0644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	log.Printf("setup: config written to %s", s.configPath)
	return nil
}
