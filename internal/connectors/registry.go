package connectors

import (
	"sort"
	"sync"

	"github.com/eblackrps/viaduct/internal/models"
)

// Factory creates a connector from a shared configuration payload.
type Factory func(Config) Connector

// Registry stores connector factories without relying on package-level mutable state.
type Registry struct {
	mu        sync.RWMutex
	factories map[models.Platform]Factory
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{
		factories: make(map[models.Platform]Factory),
	}
}

// Register stores a connector factory for the provided platform.
func (r *Registry) Register(platform models.Platform, factory Factory) {
	if r == nil || platform == "" || factory == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.factories[platform] = factory
}

// Get returns the registered connector factory for the provided platform.
func (r *Registry) Get(platform models.Platform) (Factory, bool) {
	if r == nil {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[platform]
	return factory, ok
}

// Platforms returns the registered platforms in stable sorted order.
func (r *Registry) Platforms() []models.Platform {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	platforms := make([]models.Platform, 0, len(r.factories))
	for platform := range r.factories {
		platforms = append(platforms, platform)
	}
	sort.Slice(platforms, func(i, j int) bool {
		return platforms[i] < platforms[j]
	})
	return platforms
}
