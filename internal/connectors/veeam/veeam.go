package veeam

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// VeeamConnector discovers backup inventory from Veeam Backup & Replication.
type VeeamConnector struct {
	config connectors.Config
	client *VeeamClient
}

// NewVeeamConnector creates a Veeam connector from a shared connector config.
func NewVeeamConnector(cfg connectors.Config) *VeeamConnector {
	return &VeeamConnector{config: cfg}
}

// Connect authenticates to the Veeam REST API.
func (c *VeeamConnector) Connect(ctx context.Context) error {
	client := NewVeeamClient(c.config.Address, c.config.Insecure)
	if err := client.Authenticate(ctx, c.config.Username, c.config.Password); err != nil {
		return fmt.Errorf("veeam: connect: %w", err)
	}

	c.client = client
	return nil
}

// DiscoverBackups discovers backup jobs, restore points, and repositories.
func (c *VeeamConnector) DiscoverBackups(ctx context.Context) (*models.BackupDiscoveryResult, error) {
	if c.client == nil {
		return nil, fmt.Errorf("veeam: not connected, call Connect first")
	}

	startedAt := time.Now()
	jobPayload, err := c.client.Get(ctx, "/v1/jobs")
	if err != nil {
		return nil, fmt.Errorf("veeam: discover jobs: %w", err)
	}
	restorePayload, err := c.client.Get(ctx, "/v1/objectRestorePoints")
	if err != nil {
		return nil, fmt.Errorf("veeam: discover restore points: %w", err)
	}
	repositoryPayload, err := c.client.Get(ctx, "/v1/backupInfrastructure/repositories")
	if err != nil {
		return nil, fmt.Errorf("veeam: discover repositories: %w", err)
	}

	jobsRaw, err := decodeObjectSlice(jobPayload)
	if err != nil {
		return nil, fmt.Errorf("veeam: decode jobs: %w", err)
	}
	restorePointsRaw, err := decodeObjectSlice(restorePayload)
	if err != nil {
		return nil, fmt.Errorf("veeam: decode restore points: %w", err)
	}
	repositoriesRaw, err := decodeObjectSlice(repositoryPayload)
	if err != nil {
		return nil, fmt.Errorf("veeam: decode repositories: %w", err)
	}

	result := &models.BackupDiscoveryResult{
		Jobs:          make([]models.BackupJob, 0, len(jobsRaw)),
		RestorePoints: make([]models.RestorePoint, 0, len(restorePointsRaw)),
		Repositories:  make([]models.BackupRepository, 0, len(repositoriesRaw)),
		DiscoveredAt:  time.Now().UTC(),
		Duration:      time.Since(startedAt),
	}

	for _, raw := range jobsRaw {
		result.Jobs = append(result.Jobs, mapJob(raw))
	}
	for _, raw := range restorePointsRaw {
		result.RestorePoints = append(result.RestorePoints, mapRestorePoint(raw))
	}
	for _, raw := range repositoriesRaw {
		result.Repositories = append(result.Repositories, mapRepository(raw))
	}

	return result, nil
}

// Close clears the authenticated client reference.
func (c *VeeamConnector) Close() error {
	c.client = nil
	return nil
}

func decodeObjectSlice(payload []byte) ([]map[string]interface{}, error) {
	var items []map[string]interface{}
	if err := json.Unmarshal(payload, &items); err == nil {
		return items, nil
	}

	var envelope struct {
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, err
	}

	return envelope.Data, nil
}
