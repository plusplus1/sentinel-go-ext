package reg

import (
	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/flow"
)

type AppInfo struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Desc string `json:"desc,omitempty"`
	Env  string `json:"env,omitempty"`
	Type string `json:"type,omitempty"`

	// connect parameters
	Endpoints []string          `json:"endpoints,omitempty"`
	Args      map[string]string `json:"args,omitempty"`
}

type ISentinelConfigSource interface {
	ListFlowRules(resource ...string) ([]flow.Rule, error)
	ListCircuitbreakerRules(resource ...string) ([]circuitbreaker.Rule, error)

	SaveOrUpdateFlowRule(rule flow.Rule) error
	SaveOrUpdateCircuitbreakerRule(rule circuitbreaker.Rule) error

	DeleteFlowRule(rule flow.Rule) error
	DeleteCircuitbreakerRule(rule circuitbreaker.Rule) error
}

type SentinelConfigSourceBuilder func(info AppInfo) ISentinelConfigSource
