package reg

import (
	"sync"
)

var (
	//empty     = struct{}{}
	supported = map[string]SentinelConfigSourceBuilder{}
	mu        = sync.RWMutex{}
)

func Reg(sourceType string, builder SentinelConfigSourceBuilder) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := supported[sourceType]; !ok {
		supported[sourceType] = builder
	}
}

func SourceBuilder(source string) SentinelConfigSourceBuilder {
	mu.RLock()
	defer mu.RUnlock()

	if len(supported) > 0 {
		return supported[source]
	}
	return nil
}
