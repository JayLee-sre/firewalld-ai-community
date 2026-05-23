package dashboard

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"zhiyuwaf/internal/model"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type UpdatePasswordRequest struct {
	Password string `json:"password"`
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.store.ListUsers()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	// Strip password hashes from response
	type userResp struct {
		ID        string    `json:"id"`
		Username  string    `json:"username"`
		Role      string    `json:"role"`
		CreatedAt time.Time `json:"created_at"`
	}
	resp := make([]userResp, len(users))
	for i, u := range users {
		resp[i] = userResp{ID: u.ID, Username: u.Username, Role: u.Role, CreatedAt: u.CreatedAt}
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Username == "" || len(req.Username) < 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "username must be at least 3 characters"})
		return
	}
	if len(req.Password) < minPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 12 characters"})
		return
	}
	if req.Role != "admin" && req.Role != "operator" && req.Role != "viewer" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "role must be admin, operator, or viewer"})
		return
	}

	// Check if username already exists
	existing, _ := s.store.GetUserByUsername(req.Username)
	if existing != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "username already exists"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	user := model.User{
		ID:           uuid.New().String(),
		Username:     req.Username,
		PasswordHash: string(hash),
		Role:         req.Role,
	}
	if err := s.store.CreateUser(user); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to create user: " + err.Error()})
		return
	}

	s.recordAudit("admin", dashboardClientIP(r), "user_create", "success", "username="+req.Username+" role="+req.Role)
	log.Printf("user created: %s (role: %s)", req.Username, req.Role)

	writeJSON(w, http.StatusCreated, map[string]string{"id": user.ID, "status": "created"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing user id"})
		return
	}

	// Prevent deleting the last admin
	users, _ := s.store.ListUsers()
	adminCount := 0
	for _, u := range users {
		if u.Role == "admin" {
			adminCount++
		}
	}
	if adminCount <= 1 {
		for _, u := range users {
			if u.ID == id && u.Role == "admin" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "cannot delete the last admin user"})
				return
			}
		}
	}

	if err := s.store.DeleteUser(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to delete user"})
		return
	}

	s.recordAudit("admin", dashboardClientIP(r), "user_delete", "success", "user_id="+id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleUpdateUserPassword(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing user id"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req UpdatePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
		return
	}

	if len(req.Password) < minPasswordLength {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "password must be at least 12 characters"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to hash password"})
		return
	}

	if err := s.store.UpdateUserPassword(id, string(hash)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to update password"})
		return
	}

	s.recordAudit("admin", dashboardClientIP(r), "user_password_change", "success", "user_id="+id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "password_updated"})
}
