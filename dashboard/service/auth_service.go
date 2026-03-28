package service

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/plusplus1/sentinel-go-ext/dashboard/model"
)

type AuthService struct {
	db       *sql.DB
	sessions SessionStore
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		db:       db,
		sessions: NewMySQLSessionStore(db),
	}
}

// Login authenticates a user with email and password
func (s *AuthService) Login(email, password string) (*model.User, string, error) {
	var user model.User
	var passwordHash string
	err := s.db.QueryRow(
		`SELECT id, user_id, email, name, COALESCE(avatar_url,''), password_hash, 
		 role, COALESCE(feishu_user_id,''), status 
		 FROM users WHERE email = ? AND status = 'active'`, email,
	).Scan(&user.ID, &user.UserID, &user.Email, &user.Name, &user.AvatarURL,
		&passwordHash, &user.Role, &user.FeishuUserID, &user.Status)
	if err != nil {
		log.Printf("Login failed for email %s: user not found or inactive", email)
		return nil, "", fmt.Errorf("账号或密码错误")
	}

	// Verify password
	if !verifyPassword(password, passwordHash) {
		log.Printf("Login failed for email %s: incorrect password", email)
		return nil, "", fmt.Errorf("账号或密码错误")
	}

	token, err := s.sessions.CreateSession(user.UserID, 24*time.Hour)
	if err != nil {
		log.Printf("Login failed for email %s: failed to create session - %v", email, err)
		return nil, "", fmt.Errorf("生成token失败")
	}

	// Update last login
	_, err = s.db.Exec("UPDATE users SET last_login_at = NOW() WHERE user_id = ?", user.UserID)
	if err != nil {
		log.Printf("Failed to update last login time for user %s: %v", user.UserID, err)
		// Continue with login even if update fails
	}

	log.Printf("Login successful for email %s, user ID %s", email, user.UserID)
	return &user, token, nil
}

// CreateUser creates a new user
func (s *AuthService) CreateUser(email, name, password, role string) (*model.User, error) {
	userID, err := generateUserID(s.db)
	if err != nil {
		return nil, err
	}

	hash := hashPassword(password)

	_, err = s.db.Exec(
		`INSERT INTO users (user_id, email, name, password_hash, role, status) 
		 VALUES (?, ?, ?, ?, ?, 'active')`,
		userID, email, name, hash, role)
	if err != nil {
		return nil, fmt.Errorf("创建用户失败: %v", err)
	}

	return &model.User{
		UserID: userID,
		Email:  email,
		Name:   name,
		Role:   role,
		Status: "active",
	}, nil
}

// GetUserByID gets a user by user_id
func (s *AuthService) GetUserByID(userID string) (*model.User, error) {
	var user model.User
	err := s.db.QueryRow(
		`SELECT id, user_id, email, name, COALESCE(avatar_url,''), 
		 role, COALESCE(feishu_user_id,''), status 
		 FROM users WHERE user_id = ? AND status = 'active'`, userID,
	).Scan(&user.ID, &user.UserID, &user.Email, &user.Name, &user.AvatarURL,
		&user.Role, &user.FeishuUserID, &user.Status)
	if err != nil {
		log.Printf("Error getting user by ID %s: %v", userID, err)
		return nil, fmt.Errorf("用户不存在")
	}
	return &user, nil
}

// ListUsers lists all users
func (s *AuthService) ListUsers() ([]*model.User, error) {
	rows, err := s.db.Query(
		`SELECT id, user_id, email, name, COALESCE(avatar_url,''), 
		 role, status, created_at 
		 FROM users WHERE status = 'active' ORDER BY created_at DESC`)
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.UserID, &u.Email, &u.Name, &u.AvatarURL,
			&u.Role, &u.Status, &u.CreatedAt); err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}
		users = append(users, &u)
	}

	// Check for any error during iteration
	if err := rows.Err(); err != nil {
		log.Printf("Error iterating user rows: %v", err)
		return nil, err
	}

	return users, nil
}

// CheckPermission checks if a user has permission on a resource
func (s *AuthService) CheckPermission(userID, resourceType, resourceID string) (bool, error) {
	// First, check if the user is a super admin
	var user model.User
	err := s.db.QueryRow(
		`SELECT role FROM users WHERE user_id = ? AND status = 'active'`, userID,
	).Scan(&user.Role)
	if err != nil {
		return false, err
	}

	// Super admin has full permissions
	if user.Role == "super_admin" {
		return true, nil
	}

	// For line_admin and member, check specific permissions
	var count int

	// Check if user has direct permission on this resource
	err = s.db.QueryRow(
		`SELECT COUNT(*) FROM user_permissions 
		 WHERE user_id = ? AND resource_type = ? AND resource_id = ?`,
		userID, resourceType, resourceID).Scan(&count)
	if err != nil {
		return false, err
	}

	if count > 0 {
		return true, nil
	}

	// For line_admin, check if they have permission on the parent business line
	if user.Role == "line_admin" && resourceType != "business_line" {
		// Check if resource belongs to a business line that the user manages
		var businessLineID string
		var hasLinePermission bool

		if resourceType == "app" {
			// Get business line ID from business_line_apps (app_id instead of app_key)
			err = s.db.QueryRow(
				`SELECT business_line_id FROM business_line_apps WHERE app_id = ?`,
				resourceID).Scan(&businessLineID)
		} else if resourceType == "module" || resourceType == "group" {
			// Get app_id from groups, then business line from business_line_apps
			var appID string
			err = s.db.QueryRow(
				"SELECT app_id FROM business_line_app_groups WHERE id = ?",
				resourceID).Scan(&appID)
			if err == nil {
				err = s.db.QueryRow(
					`SELECT business_line_id FROM business_line_apps WHERE app_id = ?`,
					appID).Scan(&businessLineID)
			}
		} else if resourceType == "resource" {
			// Get group ID from resources, then app_id from groups, then business line
			var groupID string
			var appID string
			err = s.db.QueryRow(
				`SELECT group_id FROM business_line_resources WHERE id = ?`,
				resourceID).Scan(&groupID)
			if err == nil {
				err = s.db.QueryRow(
					"SELECT app_id FROM business_line_app_groups WHERE id = ?",
					groupID).Scan(&appID)
				if err == nil {
					err = s.db.QueryRow(
						`SELECT business_line_id FROM business_line_apps WHERE app_id = ?`,
						appID).Scan(&businessLineID)
				}
			}
		}

		// Check if user has admin role on the business line
		if err == nil && businessLineID != "" {
			err = s.db.QueryRow(
				`SELECT COUNT(*) FROM user_permissions 
				 WHERE user_id = ? AND resource_type = 'business_line' 
				 AND resource_id = ? AND role = 'admin'`,
				userID, businessLineID).Scan(&hasLinePermission)
			if err == nil && hasLinePermission {
				return true, nil
			}
		}
	}

	// No permission found
	return false, nil
}

// GrantPermission grants a permission to a user
func (s *AuthService) GrantPermission(userID, resourceType, resourceID, role, grantedBy string) error {
	_, err := s.db.Exec(
		`INSERT INTO user_permissions (user_id, resource_type, resource_id, role, granted_by)
		 VALUES (?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE role = VALUES(role), granted_by = VALUES(granted_by)`,
		userID, resourceType, resourceID, role, grantedBy)
	return err
}

// RevokePermission revokes a permission
func (s *AuthService) RevokePermission(userID, resourceType, resourceID string) error {
	_, err := s.db.Exec(
		`DELETE FROM user_permissions WHERE user_id = ? AND resource_type = ? AND resource_id = ?`,
		userID, resourceType, resourceID)
	return err
}

// ValidateSession validates a session token and returns the user if valid
func (s *AuthService) ValidateSession(token string) (*model.User, error) {
	info, err := s.sessions.ValidateSession(token)
	if err != nil {
		return nil, err
	}
	return s.GetUserByID(info.UserID)
}

// CreateSessionForUser creates a new session for a user with the given TTL
func (s *AuthService) CreateSessionForUser(userID string, ttl time.Duration) (string, error) {
	return s.sessions.CreateSession(userID, ttl)
}

// DeleteSessionByToken deletes a session by its token
func (s *AuthService) DeleteSessionByToken(token string) error {
	return s.sessions.DeleteSession(token)
}

// CleanExpiredSessions removes all expired sessions
func (s *AuthService) CleanExpiredSessions() error {
	return s.sessions.CleanExpired()
}

// LogAudit logs an audit entry
func (s *AuthService) LogAudit(userID, action, resourceType, resourceID, detail, ip string) {
	s.db.Exec(
		`INSERT INTO user_audit_logs (user_id, action, resource_type, resource_id, detail, ip_address) 
		 VALUES (?, ?, ?, ?, ?, ?)`,
		userID, action, resourceType, resourceID, detail, ip)
}

// Helper functions
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// In case of error, return a dummy hash to prevent timing attacks
		return "invalid_hash"
	}
	return string(hash)
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateUserID(db *sql.DB) (string, error) {
	const maxAttempts = 5
	for i := 0; i < maxAttempts; i++ {
		b := make([]byte, 8)
		if _, err := rand.Read(b); err != nil {
			return "", err
		}
		userID := "u_" + hex.EncodeToString(b)

		// Check if userID already exists
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE user_id = ?)", userID).Scan(&exists)
		if err != nil {
			return "", err
		}

		if !exists {
			return userID, nil
		}
	}
	return "", fmt.Errorf("failed to generate unique user ID after %d attempts", maxAttempts)
}
