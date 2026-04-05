package veeam

import (
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func mapJob(raw map[string]interface{}) models.BackupJob {
	return models.BackupJob{
		ID:            stringValue(raw, "id"),
		Name:          stringValue(raw, "name"),
		Type:          stringValue(raw, "type"),
		Schedule:      stringValue(raw, "schedule"),
		TargetRepo:    stringValue(raw, "targetRepo"),
		RetentionDays: intValue(raw, "retentionDays"),
		ProtectedVMs:  stringSlice(raw["protectedVMs"]),
		LastRun:       timeValue(raw, "lastRun"),
		LastResult:    stringValue(raw, "lastResult"),
		Enabled:       boolValue(raw, "enabled"),
	}
}

func mapRestorePoint(raw map[string]interface{}) models.RestorePoint {
	return models.RestorePoint{
		ID:        stringValue(raw, "id"),
		VMID:      stringValue(raw, "vmId"),
		VMName:    stringValue(raw, "vmName"),
		JobName:   stringValue(raw, "jobName"),
		CreatedAt: timeValue(raw, "createdAt"),
		SizeMB:    int64Value(raw, "sizeMB"),
		Type:      stringValue(raw, "type"),
	}
}

func mapRepository(raw map[string]interface{}) models.BackupRepository {
	return models.BackupRepository{
		ID:         stringValue(raw, "id"),
		Name:       stringValue(raw, "name"),
		Type:       stringValue(raw, "type"),
		CapacityMB: int64Value(raw, "capacityMB"),
		FreeMB:     int64Value(raw, "freeMB"),
		UsedMB:     int64Value(raw, "usedMB"),
	}
}

func stringValue(raw map[string]interface{}, key string) string {
	value, _ := raw[key].(string)
	return value
}

func stringSlice(raw interface{}) []string {
	items, ok := raw.([]interface{})
	if !ok {
		return nil
	}

	out := make([]string, 0, len(items))
	for _, item := range items {
		if value, ok := item.(string); ok {
			out = append(out, value)
		}
	}

	return out
}

func intValue(raw map[string]interface{}, key string) int {
	return int(int64Value(raw, key))
}

func int64Value(raw map[string]interface{}, key string) int64 {
	switch value := raw[key].(type) {
	case float64:
		return int64(value)
	case int64:
		return value
	case int:
		return int64(value)
	default:
		return 0
	}
}

func boolValue(raw map[string]interface{}, key string) bool {
	value, _ := raw[key].(bool)
	return value
}

func timeValue(raw map[string]interface{}, key string) time.Time {
	value, _ := raw[key].(string)
	parsed, _ := time.Parse(time.RFC3339, value)
	return parsed
}
