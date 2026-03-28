package model

import "time"

// User represents a user in the system
type User struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	AvatarURL    string     `json:"avatar_url"`
	PasswordHash string     `json:"-"`    // Never expose in JSON
	Role         string     `json:"role"` // super_admin, line_admin, member
	FeishuUserID string     `json:"feishu_user_id,omitempty"`
	Status       string     `json:"status"` // active, disabled
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BusinessLine represents a business line
type BusinessLine struct {
	ID          string              `json:"id"`
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Status      string              `json:"status"` // active, deleted
	OwnerID     string              `json:"owner_id"`
	Admins      []BusinessLineAdmin `json:"admins,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

// BusinessLineAdmin represents an admin of a business line
type BusinessLineAdmin struct {
	ID             string    `json:"id"`
	BusinessLineID string    `json:"business_line_id"`
	UserID         string    `json:"user_id"`
	UserName       string    `json:"user_name,omitempty"`
	UserEmail      string    `json:"user_email,omitempty"`
	UserStatus     string    `json:"user_status,omitempty"`
	AddedBy        string    `json:"added_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// BusinessLineMember represents a member of a business line
type BusinessLineMember struct {
	ID             string    `json:"id"`
	BusinessLineID string    `json:"business_line_id"`
	UserID         string    `json:"user_id"`
	UserName       string    `json:"user_name,omitempty"`
	UserEmail      string    `json:"user_email,omitempty"`
	UserStatus     string    `json:"user_status,omitempty"`
	AddedBy        string    `json:"added_by"`
	CreatedAt      time.Time `json:"created_at"`
}

// BusinessLineApp represents an app under a business line
type BusinessLineApp struct {
	ID             string    `json:"id"`
	BusinessLineID string    `json:"business_line_id"`
	AppKey         string    `json:"app_key"`
	AppName        string    `json:"app_name"`
	EtcdURL        string    `json:"etcd_url"`
	Description    string    `json:"description"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// UserPermission represents a user's permission on a resource
type UserPermission struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	ResourceType string    `json:"resource_type"` // business_line, app, module, group
	ResourceID   string    `json:"resource_id"`
	Role         string    `json:"role"` // admin, member, viewer, owner
	GrantedBy    string    `json:"granted_by"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserAuditLog represents an audit log entry
type UserAuditLog struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Detail       string    `json:"detail"`
	IPAddress    string    `json:"ip_address"`
	CreatedAt    time.Time `json:"created_at"`
}

// UserToken represents an API token
type UserToken struct {
	ID         string     `json:"id"`
	UserID     string     `json:"user_id"`
	TokenName  string     `json:"token_name"`
	TokenHash  string     `json:"-"`
	Scopes     string     `json:"scopes"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}
