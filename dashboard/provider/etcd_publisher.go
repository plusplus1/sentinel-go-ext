package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// EtcdRulePublisher publishes rules to etcd.
type EtcdRulePublisher struct {
	client *clientv3.Client
}

// NewEtcdRulePublisher creates an EtcdRulePublisher backed by the given etcd client.
func NewEtcdRulePublisher(client *clientv3.Client) *EtcdRulePublisher {
	return &EtcdRulePublisher{client: client}
}

// PublishRules writes the JSON-encoded rule data to the given etcd key.
func (p *EtcdRulePublisher) PublishRules(key string, data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := p.client.Put(ctx, key, string(data))
	return err
}

// EtcdClientManager manages cached etcd clients per app.
type EtcdClientManager struct {
	clients map[string]*clientv3.Client
	mu      sync.Mutex
}

// NewEtcdClientManager creates a new EtcdClientManager.
func NewEtcdClientManager() *EtcdClientManager {
	return &EtcdClientManager{
		clients: make(map[string]*clientv3.Client),
	}
}

// GetOrCreateClient returns a cached or newly created etcd client for the given app.
func (m *EtcdClientManager) GetOrCreateClient(appID string, settingsJSON string) (*clientv3.Client, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("app_%s", appID)
	if client, ok := m.clients[key]; ok {
		return client, nil
	}

	endpoints := ParseEtcdEndpoints(settingsJSON)
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to etcd: %v", err)
	}

	m.clients[key] = client
	return client, nil
}

// ParseEtcdEndpoints parses etcd endpoints from settings JSON.
// Returns default endpoint if settings is empty or invalid.
func ParseEtcdEndpoints(settingsJSON string) []string {
	endpoints := []string{"http://127.0.0.1:2379"}
	if settingsJSON == "" || settingsJSON == "{}" {
		return endpoints
	}
	var settings struct {
		URL string `json:"url"`
	}
	if json.Unmarshal([]byte(settingsJSON), &settings) == nil && settings.URL != "" {
		if u, e := url.Parse(settings.URL); e == nil && u.Host != "" {
			parsed := strings.Split(u.Host, ",")
			eps := make([]string, len(parsed))
			for i, ep := range parsed {
				eps[i] = "http://" + ep
			}
			return eps
		}
	}
	return endpoints
}
