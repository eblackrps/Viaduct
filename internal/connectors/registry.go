package connectors

import (
	"sync"

	"github.com/eblackrps/viaduct/internal/models"
)

var (
	registryMu sync.RWMutex
	registry   = map[models.Platform]func(Config) Connector{}
)

// Register stores a connector factory for the provided platform.
func Register(platform models.Platform, factory func(Config) Connector) {
	if platform == "" || factory == nil {
		return
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	registry[platform] = factory
}

// Get returns the registered connector factory for the provided platform.
func Get(platform models.Platform) (func(Config) Connector, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()

	factory, ok := registry[platform]
	return factory, ok
}
