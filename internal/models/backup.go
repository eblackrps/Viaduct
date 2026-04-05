package models

import "time"

// BackupJob represents a Veeam or platform backup job protecting one or more VMs.
type BackupJob struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Schedule      string    `json:"schedule"`
	TargetRepo    string    `json:"target_repo"`
	RetentionDays int       `json:"retention_days"`
	ProtectedVMs  []string  `json:"protected_vms"`
	LastRun       time.Time `json:"last_run"`
	LastResult    string    `json:"last_result"`
	Enabled       bool      `json:"enabled"`
}

// RestorePoint represents a restorable VM backup point in time.
type RestorePoint struct {
	ID        string    `json:"id"`
	VMID      string    `json:"vm_id"`
	VMName    string    `json:"vm_name"`
	JobName   string    `json:"job_name"`
	CreatedAt time.Time `json:"created_at"`
	SizeMB    int64     `json:"size_mb"`
	Type      string    `json:"type"`
}

// BackupRepository represents a backup storage repository and its capacity.
type BackupRepository struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	CapacityMB int64  `json:"capacity_mb"`
	FreeMB     int64  `json:"free_mb"`
	UsedMB     int64  `json:"used_mb"`
}

// BackupDiscoveryResult captures backup inventory discovered from Veeam or similar systems.
type BackupDiscoveryResult struct {
	Jobs          []BackupJob        `json:"jobs"`
	RestorePoints []RestorePoint     `json:"restore_points"`
	Repositories  []BackupRepository `json:"repositories"`
	DiscoveredAt  time.Time          `json:"discovered_at"`
	Duration      time.Duration      `json:"duration"`
	Errors        []string           `json:"errors,omitempty"`
}
