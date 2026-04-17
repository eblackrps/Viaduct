package kvm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// KVMConnector discovers VM inventory from fixture or exported libvirt XML data.
type KVMConnector struct {
	config    connectors.Config
	connected bool
}

// NewKVMConnector creates a KVM connector from shared connector configuration.
func NewKVMConnector(cfg connectors.Config) *KVMConnector {
	return &KVMConnector{config: cfg}
}

// Connect marks the connector ready for XML-backed discovery.
func (c *KVMConnector) Connect(ctx context.Context) error {
	c.connected = true
	return nil
}

// Discover retrieves domain and storage metadata from XML fixture exports.
func (c *KVMConnector) Discover(ctx context.Context) (*models.DiscoveryResult, error) {
	if !c.connected {
		return nil, fmt.Errorf("kvm: not connected, call Connect first")
	}

	startedAt := time.Now()
	discoveredAt := time.Now().UTC()
	result := &models.DiscoveryResult{
		Source:       c.config.Address,
		Platform:     models.PlatformKVM,
		VMs:          make([]models.VirtualMachine, 0),
		Networks:     make([]models.NetworkInfo, 0),
		Datastores:   make([]models.DatastoreInfo, 0),
		DiscoveredAt: discoveredAt,
	}

	if strings.TrimSpace(c.config.Address) == "" {
		result.Duration = time.Since(startedAt)
		return result, nil
	}

	info, err := os.Stat(c.config.Address)
	if err != nil {
		return nil, fmt.Errorf("kvm: stat XML source %s: %w", c.config.Address, err)
	}

	files := make([]string, 0)
	if info.IsDir() {
		entries, err := os.ReadDir(c.config.Address)
		if err != nil {
			return nil, fmt.Errorf("kvm: read XML source dir %s: %w", c.config.Address, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".xml") {
				continue
			}
			files = append(files, filepath.Join(c.config.Address, entry.Name()))
		}
		sort.Strings(files)
	} else {
		files = append(files, c.config.Address)
	}

	networkIndex := make(map[string]models.NetworkInfo)
	for _, file := range files {
		// #nosec G304 -- discovery reads the explicit XML export file paths collected from the configured source.
		payload, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("kvm: read XML file %s: %w", file, err)
		}

		switch {
		case strings.Contains(string(payload), "<domain"):
			domain, err := parseDomainXML(string(payload))
			if err != nil {
				return nil, err
			}
			vm := mapDomainToVM(domain, mapKVMPowerStateName("running"), filepath.Base(c.config.Address))
			vm.DiscoveredAt = discoveredAt
			result.VMs = append(result.VMs, vm)
			for _, nic := range vm.NICs {
				if nic.Network == "" {
					continue
				}
				networkIndex[nic.Network] = models.NetworkInfo{
					ID:     nic.Network,
					Name:   nic.Network,
					Type:   "bridge",
					Switch: nic.Network,
				}
			}
		case strings.Contains(string(payload), "<pool"):
			pool, err := parseStoragePoolXML(string(payload))
			if err != nil {
				return nil, err
			}
			result.Datastores = append(result.Datastores, models.DatastoreInfo{
				ID:   pool.Name,
				Name: pool.Name,
				Type: pool.Type,
			})
		}
	}

	for _, network := range networkIndex {
		result.Networks = append(result.Networks, network)
	}
	sort.Slice(result.Networks, func(i, j int) bool {
		return result.Networks[i].Name < result.Networks[j].Name
	})
	result.Duration = time.Since(startedAt)
	return result, nil
}

// Platform returns the KVM platform identifier.
func (c *KVMConnector) Platform() models.Platform {
	return models.PlatformKVM
}

// Close releases XML-backed discovery state.
func (c *KVMConnector) Close() error {
	c.connected = false
	return nil
}

func mapKVMPowerStateName(state string) models.PowerState {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "blocked":
		return models.PowerOn
	case "paused", "pmsuspended":
		return models.PowerSuspend
	case "shutoff", "shutdown", "crashed":
		return models.PowerOff
	default:
		return models.PowerUnknown
	}
}

var _ connectors.Connector = (*KVMConnector)(nil)
