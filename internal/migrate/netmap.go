package migrate

import (
	"fmt"

	"github.com/eblackrps/viaduct/internal/models"
)

// NetworkMapper maps source NICs onto target platform networks.
type NetworkMapper struct {
	mappings       map[string]string
	targetNetworks []models.NetworkInfo
}

// MappedNIC represents a source NIC after network remapping.
type MappedNIC struct {
	Original      models.NIC `json:"original"`
	TargetNetwork string     `json:"target_network"`
	TargetVLAN    int        `json:"target_vlan"`
}

// NewNetworkMapper creates a network mapper for a migration target.
func NewNetworkMapper(mappings map[string]string, targetNetworks []models.NetworkInfo) *NetworkMapper {
	return &NetworkMapper{
		mappings:       copyStringMap(mappings),
		targetNetworks: append([]models.NetworkInfo(nil), targetNetworks...),
	}
}

// MapNIC maps a single source NIC to a target network.
func (m *NetworkMapper) MapNIC(nic models.NIC) (*MappedNIC, error) {
	targetNetwork, ok := m.mappings[nic.Network]
	if !ok {
		return nil, fmt.Errorf("no mapping for network: %s", nic.Network)
	}

	for _, target := range m.targetNetworks {
		if target.Name == targetNetwork {
			return &MappedNIC{
				Original:      nic,
				TargetNetwork: target.Name,
				TargetVLAN:    target.VlanID,
			}, nil
		}
	}

	return nil, fmt.Errorf("target network %q does not exist", targetNetwork)
}

// MapAllNICs maps a slice of source NICs and returns all errors without stopping early.
func (m *NetworkMapper) MapAllNICs(nics []models.NIC) ([]MappedNIC, []error) {
	mapped := make([]MappedNIC, 0, len(nics))
	errs := make([]error, 0)

	for _, nic := range nics {
		item, err := m.MapNIC(nic)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		mapped = append(mapped, *item)
	}

	return mapped, errs
}

// ValidateTargetNetworks validates that every mapping points to a real target network.
func (m *NetworkMapper) ValidateTargetNetworks() []error {
	errs := make([]error, 0)
	for source, target := range m.mappings {
		found := false
		for _, network := range m.targetNetworks {
			if network.Name == target {
				found = true
				break
			}
		}
		if !found {
			errs = append(errs, fmt.Errorf("mapped network %q for source %q does not exist on target", target, source))
		}
	}

	return errs
}
