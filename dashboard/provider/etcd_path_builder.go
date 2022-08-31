package provider

import "fmt"

// EtcdPathBuilder constructs etcd key paths for Sentinel rules.
type EtcdPathBuilder struct {
	Prefix string // defaults to "/sentinel"
}

// NewEtcdPathBuilder creates an EtcdPathBuilder with the default prefix "/sentinel".
func NewEtcdPathBuilder() *EtcdPathBuilder {
	return &EtcdPathBuilder{Prefix: "/sentinel"}
}

// BuildPath constructs the full etcd key: /sentinel/{line}/{app}/{group}/{resource}/{ruleType}
func (b *EtcdPathBuilder) BuildPath(line, app, group, resource, ruleType string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s/%s", b.Prefix, line, app, group, resource, ruleType)
}
