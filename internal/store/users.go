package store

import (
	"database/sql"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

func (s *Store) ListUsers() ([]model.User, error) {
	rows, err := s.db.Query("SELECT id, username, password_hash, role, created_at FROM users ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

func (s *Store) GetUserByUsername(username string) (*model.User, error) {
	var u model.User
	err := s.db.QueryRow("SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?", username).
		Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *Store) CreateUser(u model.User) error {
	if u.ID == "" {
		u.ID = uuid.New().String()
	}
	if u.Role == "" {
		u.Role = "viewer"
	}
	now := time.Now()
	_, err := s.db.Exec(
		"INSERT INTO users (id, username, password_hash, role, created_at) VALUES (?, ?, ?, ?, ?)",
		u.ID, u.Username, u.PasswordHash, u.Role, now,
	)
	return err
}

func (s *Store) DeleteUser(id string) error {
	_, err := s.db.Exec("DELETE FROM users WHERE id = ?", id)
	return err
}

func (s *Store) UpdateUserPassword(id string, hash string) error {
	_, err := s.db.Exec("UPDATE users SET password_hash = ? WHERE id = ?", hash, id)
	return err
}
