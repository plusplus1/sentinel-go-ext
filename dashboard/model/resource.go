package model

import "time"

// Resource represents a resource metadata
type Resource struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	AppID            string    `json:"app_id"`
	GroupID          *string   `json:"group_id"`
	GroupName        string    `json:"group_name"`
	GroupDescription string    `json:"group_description"`
	Type             string    `json:"type"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// ResourceWithRules represents a resource with its rules aggregated
type ResourceWithRules struct {
	Resource  *Resource                `json:"resource"`
	Group     *Group                   `json:"group"`
	FlowRules []map[string]interface{} `json:"flow_rules"`
	CBRRules  []map[string]interface{} `json:"circuit_breaker_rules"`
	Summary   *RuleSummary             `json:"summary"`
}

// RuleSummary contains summary of rules for a resource
type RuleSummary struct {
	TotalRules     int    `json:"total_rules"`
	EnabledRules   int    `json:"enabled_rules"`
	TriggeredRules int    `json:"triggered_rules"`
	HealthStatus   string `json:"health_status"` // healthy, warning, critical
}
