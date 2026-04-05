package deps

import (
	"fmt"
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

// GraphNode represents a node in the dependency graph.
type GraphNode struct {
	ID       string            `json:"id"`
	Label    string            `json:"label"`
	Type     string            `json:"type"`
	Platform models.Platform   `json:"platform,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// GraphEdge represents a relationship between two graph nodes.
type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Label  string `json:"label"`
}

// DependencyGraph contains graph nodes and edges for dashboard visualization.
type DependencyGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// BuildGraph builds a dependency graph from inventory and backup discovery results.
func BuildGraph(inventory *models.DiscoveryResult, backups *models.BackupDiscoveryResult) *DependencyGraph {
	graph := &DependencyGraph{
		Nodes: make([]GraphNode, 0),
		Edges: make([]GraphEdge, 0),
	}
	if inventory == nil {
		return graph
	}

	nodeIndex := make(map[string]struct{})
	addNode := func(node GraphNode) {
		if _, ok := nodeIndex[node.ID]; ok {
			return
		}
		nodeIndex[node.ID] = struct{}{}
		graph.Nodes = append(graph.Nodes, node)
	}

	networkNodes := make(map[string]string)
	datastoreNodes := make(map[string]string)

	for _, vm := range inventory.VMs {
		vmNodeID := "vm:" + vm.ID
		if vm.ID == "" {
			vmNodeID = "vm:" + vm.Name
		}
		addNode(GraphNode{
			ID:       vmNodeID,
			Label:    vm.Name,
			Type:     "vm",
			Platform: vm.Platform,
			Metadata: map[string]string{"host": vm.Host, "cluster": vm.Cluster},
		})

		for _, nic := range vm.NICs {
			if nic.Network == "" {
				continue
			}
			networkNodeID, ok := networkNodes[nic.Network]
			if !ok {
				networkNodeID = "network:" + nic.Network
				networkNodes[nic.Network] = networkNodeID
				addNode(GraphNode{
					ID:       networkNodeID,
					Label:    nic.Network,
					Type:     "network",
					Metadata: map[string]string{"network": nic.Network},
				})
			}
			graph.Edges = append(graph.Edges, GraphEdge{
				Source: vmNodeID,
				Target: networkNodeID,
				Type:   "network",
				Label:  nic.Network,
			})
		}

		for _, disk := range vm.Disks {
			if disk.StorageBackend == "" {
				continue
			}
			datastoreNodeID, ok := datastoreNodes[disk.StorageBackend]
			if !ok {
				datastoreNodeID = "datastore:" + disk.StorageBackend
				datastoreNodes[disk.StorageBackend] = datastoreNodeID
				addNode(GraphNode{
					ID:       datastoreNodeID,
					Label:    disk.StorageBackend,
					Type:     "datastore",
					Metadata: map[string]string{"storage": disk.StorageBackend},
				})
			}
			graph.Edges = append(graph.Edges, GraphEdge{
				Source: vmNodeID,
				Target: datastoreNodeID,
				Type:   "storage",
				Label:  disk.Name,
			})
		}
	}

	if backups == nil {
		return graph
	}

	nameIndex := make(map[string]string, len(inventory.VMs))
	for _, vm := range inventory.VMs {
		vmNodeID := "vm:" + vm.ID
		if vm.ID == "" {
			vmNodeID = "vm:" + vm.Name
		}
		nameIndex[strings.ToLower(vm.Name)] = vmNodeID
	}

	for _, job := range backups.Jobs {
		jobNodeID := "backup:" + job.ID
		if job.ID == "" {
			jobNodeID = "backup:" + strings.ReplaceAll(strings.ToLower(job.Name), " ", "-")
		}
		addNode(GraphNode{
			ID:    jobNodeID,
			Label: job.Name,
			Type:  "backup-job",
			Metadata: map[string]string{
				"repository": job.TargetRepo,
				"schedule":   job.Schedule,
			},
		})

		for _, protectedVM := range job.ProtectedVMs {
			vmNodeID, ok := nameIndex[strings.ToLower(protectedVM)]
			if !ok {
				continue
			}
			graph.Edges = append(graph.Edges, GraphEdge{
				Source: vmNodeID,
				Target: jobNodeID,
				Type:   "backup",
				Label:  fmt.Sprintf("%s protects %s", job.Name, protectedVM),
			})
		}
	}

	return graph
}
