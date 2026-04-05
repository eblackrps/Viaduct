package kvm

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

type domainXML struct {
	XMLName xml.Name      `xml:"domain"`
	Type    string        `xml:"type,attr"`
	Name    string        `xml:"name"`
	UUID    string        `xml:"uuid"`
	Memory  memoryXML     `xml:"memory"`
	VCPU    int           `xml:"vcpu"`
	OS      domainOSXML   `xml:"os"`
	Devices domainDevices `xml:"devices"`
}

type memoryXML struct {
	Unit  string `xml:"unit,attr"`
	Value int64  `xml:",chardata"`
}

type domainOSXML struct {
	Type string `xml:"type"`
}

type domainDevices struct {
	Disks      []domainDiskXML      `xml:"disk"`
	Interfaces []domainInterfaceXML `xml:"interface"`
}

type domainDiskXML struct {
	Device string          `xml:"device,attr"`
	Driver domainDriverXML `xml:"driver"`
	Source domainSourceXML `xml:"source"`
	Target domainTargetXML `xml:"target"`
}

type domainDriverXML struct {
	Type string `xml:"type,attr"`
}

type domainSourceXML struct {
	File   string `xml:"file,attr"`
	Dev    string `xml:"dev,attr"`
	Pool   string `xml:"pool,attr"`
	Volume string `xml:"volume,attr"`
}

type domainTargetXML struct {
	Dev string `xml:"dev,attr"`
}

type domainInterfaceXML struct {
	Type   string         `xml:"type,attr"`
	MAC    domainMACXML   `xml:"mac"`
	Source ifaceSourceXML `xml:"source"`
	Target ifaceTargetXML `xml:"target"`
	Model  ifaceModelXML  `xml:"model"`
	Alias  ifaceAliasXML  `xml:"alias"`
}

type domainMACXML struct {
	Address string `xml:"address,attr"`
}

type ifaceSourceXML struct {
	Bridge  string `xml:"bridge,attr"`
	Network string `xml:"network,attr"`
}

type ifaceTargetXML struct {
	Dev string `xml:"dev,attr"`
}

type ifaceModelXML struct {
	Type string `xml:"type,attr"`
}

type ifaceAliasXML struct {
	Name string `xml:"name,attr"`
}

type storagePoolXML struct {
	XMLName xml.Name `xml:"pool"`
	Type    string   `xml:"type,attr"`
	Name    string   `xml:"name"`
	Target  struct {
		Path string `xml:"path"`
	} `xml:"target"`
}

func parseDomainXML(payload string) (*domainXML, error) {
	var domain domainXML
	if err := xml.Unmarshal([]byte(payload), &domain); err != nil {
		return nil, fmt.Errorf("parse domain XML: %w", err)
	}
	return &domain, nil
}

func parseStoragePoolXML(payload string) (*storagePoolXML, error) {
	var pool storagePoolXML
	if err := xml.Unmarshal([]byte(payload), &pool); err != nil {
		return nil, fmt.Errorf("parse storage pool XML: %w", err)
	}
	return &pool, nil
}

func mapDomainToVM(domain *domainXML, powerState models.PowerState, host string) models.VirtualMachine {
	if domain == nil {
		return models.VirtualMachine{}
	}

	disks := make([]models.Disk, 0)
	for _, disk := range domain.Devices.Disks {
		if disk.Device != "disk" {
			continue
		}

		storageBackend := disk.Source.Pool
		if storageBackend == "" {
			storageBackend = firstNonEmpty(disk.Source.File, disk.Source.Dev, disk.Source.Volume)
		}

		disks = append(disks, models.Disk{
			ID:             firstNonEmpty(disk.Target.Dev, disk.Source.File, disk.Source.Dev, disk.Source.Volume),
			Name:           firstNonEmpty(disk.Target.Dev, disk.Source.Volume, disk.Source.File),
			Thin:           strings.EqualFold(disk.Driver.Type, "qcow2"),
			StorageBackend: storageBackend,
		})
	}

	nics := make([]models.NIC, 0, len(domain.Devices.Interfaces))
	for _, iface := range domain.Devices.Interfaces {
		network := firstNonEmpty(iface.Source.Bridge, iface.Source.Network)
		nics = append(nics, models.NIC{
			ID:         firstNonEmpty(iface.Alias.Name, iface.Target.Dev, iface.MAC.Address),
			Name:       firstNonEmpty(iface.Target.Dev, iface.Model.Type, iface.MAC.Address),
			MACAddress: iface.MAC.Address,
			Network:    network,
			Connected:  true,
		})
	}

	return models.VirtualMachine{
		ID:           firstNonEmpty(domain.UUID, domain.Name),
		Name:         domain.Name,
		Platform:     models.PlatformKVM,
		PowerState:   powerState,
		CPUCount:     domain.VCPU,
		MemoryMB:     memoryToMB(domain.Memory),
		Disks:        disks,
		NICs:         nics,
		GuestOS:      domain.OS.Type,
		Host:         host,
		DiscoveredAt: time.Now().UTC(),
		SourceRef:    firstNonEmpty(domain.UUID, domain.Name),
	}
}

func memoryToMB(memory memoryXML) int {
	switch strings.ToLower(memory.Unit) {
	case "g", "gb", "gib":
		return int(memory.Value * 1024)
	case "m", "mb", "mib", "":
		return int(memory.Value)
	case "k", "kb", "kib":
		return int(memory.Value / 1024)
	case "b":
		return int(memory.Value / (1024 * 1024))
	default:
		if parsed, err := strconv.Atoi(memory.Unit); err == nil {
			return int(memory.Value / int64(parsed))
		}
		return int(memory.Value)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
