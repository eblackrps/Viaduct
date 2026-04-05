package veeam

import (
	"strings"

	"github.com/eblackrps/viaduct/internal/models"
)

// CorrelateBackups correlates discovered backup jobs to inventory VMs by VM name.
func CorrelateBackups(inventory *models.DiscoveryResult, backups *models.BackupDiscoveryResult) map[string][]models.BackupJob {
	correlated := make(map[string][]models.BackupJob)
	if inventory == nil || backups == nil {
		return correlated
	}

	nameIndex := make(map[string]string, len(inventory.VMs))
	for _, vm := range inventory.VMs {
		nameIndex[strings.ToLower(vm.Name)] = vm.ID
	}

	for _, job := range backups.Jobs {
		for _, protectedVM := range job.ProtectedVMs {
			vmID, ok := nameIndex[strings.ToLower(protectedVM)]
			if !ok {
				continue
			}
			correlated[vmID] = append(correlated[vmID], job)
		}
	}

	return correlated
}
