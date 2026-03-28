package model

import "time"

// Group represents a resource group
type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AppID       string    `json:"app_id"`
	ParentID    *string   `json:"parent_id"`
	IsDefault   bool      `json:"is_default"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Tags        []string  `json:"tags"`
}

// GroupMember represents a resource in a group
type GroupMember struct {
	GroupID    string `json:"group_id"`
	ResourceID string `json:"resource_id"`
}

// GroupWithMembers represents a group with its members
type GroupWithMembers struct {
	Group   *Group    `json:"group"`
	Members []string  `json:"members"`
}
