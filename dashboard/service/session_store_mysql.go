package service

import (
	"database/sql"
	"fmt"
	"time"
)

type MySQLSessionStore struct {
	db *sql.DB
}

func NewMySQLSessionStore(db *sql.DB) *MySQLSessionStore {
	return &MySQLSessionStore{db: db}
}

func (s *MySQLSessionStore) CreateSession(userID string, ttl time.Duration) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	expiresAt := time.Now().Add(ttl)
	_, err = s.db.Exec(
		`INSERT INTO user_tokens (user_id, token, expires_at)
		 VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE token = VALUES(token), expires_at = VALUES(expires_at)`,
		userID, token, expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}

	return token, nil
}

func (s *MySQLSessionStore) ValidateSession(token string) (*SessionInfo, error) {
	var info SessionInfo
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM user_tokens
		 WHERE token = ? AND expires_at > NOW()`,
		token).Scan(&info.UserID, &info.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("会话不存在或已过期")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to validate session: %v", err)
	}
	return &info, nil
}

func (s *MySQLSessionStore) DeleteSession(token string) error {
	_, err := s.db.Exec("DELETE FROM user_tokens WHERE token = ?", token)
	if err != nil {
		return fmt.Errorf("failed to delete session: %v", err)
	}
	return nil
}

func (s *MySQLSessionStore) DeleteUserSessions(userID string) error {
	_, err := s.db.Exec("DELETE FROM user_tokens WHERE user_id = ?", userID)
	if err != nil {
		return fmt.Errorf("failed to delete user sessions: %v", err)
	}
	return nil
}

func (s *MySQLSessionStore) CleanExpired() error {
	_, err := s.db.Exec("DELETE FROM user_tokens WHERE expires_at < NOW()")
	if err != nil {
		return fmt.Errorf("failed to clean expired sessions: %v", err)
	}
	return nil
}
