package service

import "time"

// SessionStore defines the interface for session persistence
type SessionStore interface {
	CreateSession(userID string, ttl time.Duration) (token string, err error)
	ValidateSession(token string) (*SessionInfo, error)
	DeleteSession(token string) error
	DeleteUserSessions(userID string) error
	CleanExpired() error
}

// SessionInfo holds session metadata
type SessionInfo struct {
	UserID    string
	ExpiresAt time.Time
}
