package provider

// RulePublisher publishes rules to a config center (etcd, Nacos, etc.)
type RulePublisher interface {
	// PublishRules publishes a set of rules to the config center
	// key is the full path (e.g., /sentinel/line/app/group/resource/flow)
	// data is the JSON-encoded rule array
	PublishRules(key string, data []byte) error
}

// RulePathBuilder constructs config center key paths
type RulePathBuilder interface {
	// BuildPath constructs: /sentinel/{line}/{app}/{group}/{resource}/{ruleType}
	BuildPath(line, app, group, resource, ruleType string) string
}
