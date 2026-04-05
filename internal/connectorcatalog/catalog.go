package connectorcatalog

import (
	"fmt"
	"strings"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/connectors/hyperv"
	"github.com/eblackrps/viaduct/internal/connectors/kvm"
	"github.com/eblackrps/viaduct/internal/connectors/nutanix"
	"github.com/eblackrps/viaduct/internal/connectors/plugin"
	"github.com/eblackrps/viaduct/internal/connectors/proxmox"
	"github.com/eblackrps/viaduct/internal/connectors/vmware"
	"github.com/eblackrps/viaduct/internal/models"
)

// Catalog provides connector factory lookup for built-in and plugin-backed platforms.
type Catalog struct {
	registry *connectors.Registry
	host     *plugin.PluginHost
}

// New creates a connector catalog with built-in connectors and optional plugin connectors.
func New(pluginPaths map[string]string) (*Catalog, error) {
	registry := connectors.NewRegistry()
	registerBuiltIns(registry)

	var host *plugin.PluginHost
	if len(pluginPaths) > 0 {
		host = plugin.NewPluginHost()
		for rawPlatform, rawPath := range pluginPaths {
			platform := models.Platform(strings.ToLower(strings.TrimSpace(rawPlatform)))
			path := strings.TrimSpace(rawPath)
			if platform == "" || path == "" {
				continue
			}

			if err := host.LoadPlugin(path); err != nil {
				host.ShutdownAll()
				return nil, fmt.Errorf("load plugin for %s: %w", platform, err)
			}

			factory, err := host.Factory(string(platform))
			if err != nil {
				host.ShutdownAll()
				return nil, fmt.Errorf("register plugin for %s: %w", platform, err)
			}
			registry.Register(platform, factory)
		}
	}

	return &Catalog{
		registry: registry,
		host:     host,
	}, nil
}

// Build returns a connector instance for the requested platform and config.
func (c *Catalog) Build(platform models.Platform, cfg connectors.Config) (connectors.Connector, error) {
	if c == nil || c.registry == nil {
		return nil, fmt.Errorf("connector catalog is not configured")
	}

	factory, ok := c.registry.Get(platform)
	if !ok {
		return nil, fmt.Errorf("unsupported platform %q", platform)
	}
	return factory(cfg), nil
}

// Close shuts down any plugin-backed connector processes managed by the catalog.
func (c *Catalog) Close() {
	if c == nil || c.host == nil {
		return
	}
	c.host.ShutdownAll()
}

// Platforms returns the registered connector platforms in stable order.
func (c *Catalog) Platforms() []models.Platform {
	if c == nil || c.registry == nil {
		return nil
	}
	return c.registry.Platforms()
}

func registerBuiltIns(registry *connectors.Registry) {
	registry.Register(models.PlatformVMware, func(cfg connectors.Config) connectors.Connector {
		return vmware.NewVMwareConnector(cfg)
	})
	registry.Register(models.PlatformProxmox, func(cfg connectors.Config) connectors.Connector {
		return proxmox.NewProxmoxConnector(cfg)
	})
	registry.Register(models.PlatformHyperV, func(cfg connectors.Config) connectors.Connector {
		return hyperv.NewHyperVConnector(cfg)
	})
	registry.Register(models.PlatformKVM, func(cfg connectors.Config) connectors.Connector {
		return kvm.NewKVMConnector(cfg)
	})
	registry.Register(models.PlatformNutanix, func(cfg connectors.Config) connectors.Connector {
		return nutanix.NewNutanixConnector(cfg)
	})
}
