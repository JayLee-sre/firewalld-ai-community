package dashboard

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type loginAttempt struct {
	count   int
	lockEnd time.Time
	lastTry time.Time
}

var (
	loginAttempts = make(map[string]*loginAttempt)
	loginMu       sync.Mutex
)

const (
	maxLoginAttempts  = 5
	lockDuration      = 15 * time.Minute
	attemptWindow     = 5 * time.Minute
	minPasswordLength = 12
)

func checkLoginRateLimit(ip string) bool {
	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		return true
	}

	now := time.Now()
	// Check if currently locked
	if now.Before(attempt.lockEnd) {
		return false
	}
	// Reset if window expired
	if now.Sub(attempt.lastTry) > attemptWindow {
		attempt.count = 0
	}
	return true
}

func recordLoginFailure(ip string) (locked bool) {
	loginMu.Lock()
	defer loginMu.Unlock()

	attempt, exists := loginAttempts[ip]
	if !exists {
		attempt = &loginAttempt{}
		loginAttempts[ip] = attempt
	}

	now := time.Now()
	if now.Sub(attempt.lastTry) > attemptWindow {
		attempt.count = 1
	} else {
		attempt.count++
	}
	attempt.lastTry = now

	if attempt.count >= maxLoginAttempts {
		attempt.lockEnd = now.Add(lockDuration)
		return true
	}
	return false
}

func clearLoginAttempts(ip string) {
	loginMu.Lock()
	defer loginMu.Unlock()
	delete(loginAttempts, ip)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	clientIP := dashboardClientIP(r)

	if !checkLoginRateLimit(clientIP) {
		s.recordAudit("admin", clientIP, "login", "blocked", "too many failed attempts")
		http.Error(w, `{"error":"too many failed attempts, please try again later"}`, http.StatusTooManyRequests)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	username := req.Username
	user, err := s.store.GetUserByUsername(username)
	if err != nil || user == nil {
		// Fallback: check legacy admin_password_hash for migration
		if username == "admin" {
			storedHash, _ := s.store.GetSetting("admin_password_hash")
			if storedHash != "" && bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)) == nil {
				clearLoginAttempts(clientIP)
				s.recordAudit(username, clientIP, "login", "success", "dashboard login (legacy)")
				token, err := GenerateToken(s.cfg.Dashboard.JWTSecret, username, "admin")
				if err != nil {
					http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"token": token})
				return
			}
		}
		if locked := recordLoginFailure(clientIP); locked {
			log.Printf("IP %s locked due to too many failed login attempts", clientIP)
		}
		s.recordAudit(username, clientIP, "login", "failed", "invalid username")
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		if locked := recordLoginFailure(clientIP); locked {
			log.Printf("IP %s locked due to too many failed login attempts", clientIP)
		}
		s.recordAudit(username, clientIP, "login", "failed", "invalid password")
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	clearLoginAttempts(clientIP)
	s.recordAudit(username, clientIP, "login", "success", "dashboard login")

	token, err := GenerateToken(s.cfg.Dashboard.JWTSecret, user.Username, user.Role)
	if err != nil {
		http.Error(w, `{"error":"failed to generate token"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < minPasswordLength {
		http.Error(w, `{"error":"password must be at least 12 characters"}`, http.StatusBadRequest)
		return
	}

	storedHash, _ := s.store.GetSetting("admin_password_hash")
	if storedHash == "" {
		http.Error(w, `{"error":"no password set"}`, http.StatusBadRequest)
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.OldPassword)); err != nil {
		s.recordAudit("admin", dashboardClientIP(r), "password_change", "failed", "old password incorrect")
		http.Error(w, `{"error":"old password incorrect"}`, http.StatusUnauthorized)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	if err := s.store.SetSetting("admin_password_hash", string(hash)); err != nil {
		http.Error(w, `{"error":"failed to save password"}`, http.StatusInternalServerError)
		return
	}
	s.recordAudit("admin", dashboardClientIP(r), "password_change", "success", "admin password changed")

	writeJSON(w, http.StatusOK, map[string]string{"status": "password_changed"})
}

func dashboardClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	peerIP := net.ParseIP(host)
	if peerIP == nil || !peerIP.IsLoopback() {
		return host
	}

	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if parsed := net.ParseIP(ip); parsed != nil && !parsed.IsLoopback() {
			return ip
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		if parsed := net.ParseIP(realIP); parsed != nil && !parsed.IsLoopback() {
			return realIP
		}
	}

	return host
}
