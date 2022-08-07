package etcd

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/alibaba/sentinel-golang/core/circuitbreaker"
	"github.com/alibaba/sentinel-golang/core/flow"
	"github.com/bytedance/sonic"
	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/plusplus1/sentinel-go-ext/source/reg"
)

const (
	sourceType = `etcd`
)

const (
	_gPrefix         = "/sentinel/go/rules"
	_rulePrefixFlow  = "/flow"
	_rulePrefixBreak = "/circuitbreaker"
)

var (
	pool = map[string]reg.ISentinelConfigSource{}
	mu   sync.Mutex
)

func init() {
	reg.Reg(sourceType, func(info reg.AppInfo) reg.ISentinelConfigSource {
		mu.Lock()
		defer mu.Unlock()

		if obj, ok := pool[info.Id]; ok {
			return obj
		}

		obj := newConfigSource(info)
		pool[info.Id] = obj
		return obj
	})
}

func newConfigSource(info reg.AppInfo) reg.ISentinelConfigSource {
	obj := new(sourceImpl)
	obj.endpoints = make([]string, 0, len(info.Endpoints))
	obj.args = make(map[string]string, len(info.Args))
	obj.app = info.Name
	obj.env = info.Env
	obj.baseKey = fmt.Sprintf("%s/%s/%s", _gPrefix, obj.app, obj.env)

	for _, e := range info.Endpoints {
		obj.endpoints = append(obj.endpoints, e)
	}
	for k, v := range info.Args {
		obj.args[k] = v
	}
	_ = obj.ensureClient()
	return obj
}

type sourceImpl struct {
	mu sync.Mutex

	endpoints []string
	args      map[string]string
	app       string
	env       string
	baseKey   string

	//client
	client *clientv3.Client
}

func (s *sourceImpl) ListFlowRules(resources ...string) ([]flow.Rule, error) {
	if e := s.ensureClient(); e != nil {
		return nil, e
	}

	key := s.baseKey + _rulePrefixFlow
	set := map[string]flow.Rule{}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	resp, e := s.client.Get(ctx, key, clientv3.WithPrefix())
	if e != nil {
		return nil, e
	}

	var rules []flow.Rule
	for _, kv := range resp.Kvs {
		r := flow.Rule{}
		if je := sonic.Unmarshal(kv.Value, &r); je == nil {
			if _, ok := set[r.Resource]; !ok {
				rules = append(rules, r)
				set[r.Resource] = r
			}
		}
	}

	if len(resources) > 0 && len(rules) > 0 {
		rules = rules[0:0]
		for _, r := range resources {
			if rr, ok := set[r]; ok {
				rules = append(rules, rr)
			}
		}
	}

	return rules, nil
}

func (s *sourceImpl) ListCircuitbreakerRules(resources ...string) ([]circuitbreaker.Rule, error) {

	if e := s.ensureClient(); e != nil {
		return nil, e
	}
	key := s.baseKey + _rulePrefixBreak
	set := map[string]circuitbreaker.Rule{}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	resp, e := s.client.Get(ctx, key, clientv3.WithPrefix())
	if e != nil {
		return nil, e
	}

	var rules []circuitbreaker.Rule
	for _, kv := range resp.Kvs {
		r := circuitbreaker.Rule{}
		if je := sonic.Unmarshal(kv.Value, &r); je == nil {
			if _, ok := set[r.Resource]; !ok {
				rules = append(rules, r)
				set[r.Resource] = r
			}
		}
	}

	if len(resources) > 0 && len(rules) > 0 {
		rules = rules[0:0]
		for _, name := range resources {
			if r, ok := set[name]; ok {
				rules = append(rules, r)
			}
		}
	}
	return rules, nil
}

func (s *sourceImpl) SaveOrUpdateFlowRule(rule flow.Rule) error {
	if e := s.ensureClient(); e != nil {
		return e
	}
	key := s.baseKey + _rulePrefixFlow
	key += "/" + rule.Resource
	value, _ := sonic.Marshal(rule)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if _, e := s.client.Put(ctx, key, string(value)); e != nil {
		return e
	}
	return nil
}

func (s *sourceImpl) SaveOrUpdateCircuitbreakerRule(rule circuitbreaker.Rule) error {
	if e := s.ensureClient(); e != nil {
		return e
	}
	key := s.baseKey + _rulePrefixBreak
	key += "/" + rule.Resource
	value, _ := sonic.Marshal(rule)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if _, e := s.client.Put(ctx, key, string(value)); e != nil {
		return e
	}
	return nil
}

func (s *sourceImpl) DeleteFlowRule(rule flow.Rule) error {
	if e := s.ensureClient(); e != nil {
		return e
	}
	key := s.baseKey + _rulePrefixFlow
	key += "/" + rule.Resource
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if _, e := s.client.Delete(ctx, key); e != nil {
		return e
	}
	return nil
}

func (s *sourceImpl) DeleteCircuitbreakerRule(rule circuitbreaker.Rule) error {

	if e := s.ensureClient(); e != nil {
		return e
	}
	key := s.baseKey + _rulePrefixBreak
	key += "/" + rule.Resource
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if _, e := s.client.Delete(ctx, key); e != nil {
		return e
	}
	return nil

}

func (s *sourceImpl) ensureClient() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		timeout := time.Duration(500)
		if s.args != nil {
			if t, _ := strconv.ParseInt(s.args[`timeout`], 10, 64); t > 0 {
				timeout = time.Duration(t)
			}
		}

		if c, e := clientv3.New(clientv3.Config{Endpoints: s.endpoints, DialTimeout: timeout * time.Millisecond}); e == nil {
			s.client = c
		} else {
			return e
		}
	}
	return nil

}
