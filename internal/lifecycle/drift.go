package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

// DriftType identifies the type of lifecycle drift that was detected.
type DriftType string

const (
	// DriftAdded indicates a workload appeared since the baseline.
	DriftAdded DriftType = "added"
	// DriftRemoved indicates a workload disappeared since the baseline.
	DriftRemoved DriftType = "removed"
	// DriftModified indicates a workload changed relative to the baseline.
	DriftModified DriftType = "modified"
	// DriftPolicyViolation indicates a workload violated a lifecycle policy.
	DriftPolicyViolation DriftType = "policy_violation"
)

// DriftSeverity identifies how urgent a drift event is.
type DriftSeverity string

const (
	// DriftSeverityCritical indicates urgent drift.
	DriftSeverityCritical DriftSeverity = "critical"
	// DriftSeverityWarning indicates notable but non-blocking drift.
	DriftSeverityWarning DriftSeverity = "warning"
	// DriftSeverityInfo indicates informational drift.
	DriftSeverityInfo DriftSeverity = "info"
)

// DriftEvent records a single detected drift condition.
type DriftEvent struct {
	// Type identifies the drift category.
	Type DriftType `json:"type" yaml:"type"`
	// VM is the workload associated with the event.
	VM models.VirtualMachine `json:"vm" yaml:"vm"`
	// Field is the field that changed or violated policy.
	Field string `json:"field,omitempty" yaml:"field,omitempty"`
	// OldValue is the baseline value when available.
	OldValue interface{} `json:"old_value,omitempty" yaml:"old_value,omitempty"`
	// NewValue is the current value when available.
	NewValue interface{} `json:"new_value,omitempty" yaml:"new_value,omitempty"`
	// Severity is the event severity.
	Severity DriftSeverity `json:"severity" yaml:"severity"`
	// DetectedAt is when the event was generated.
	DetectedAt time.Time `json:"detected_at" yaml:"detected_at"`
}

// DriftConfig configures change thresholds and webhook behavior.
type DriftConfig struct {
	// IgnoreFields skips matching field names during comparison.
	IgnoreFields []string `json:"ignore_fields,omitempty" yaml:"ignore_fields,omitempty"`
	// MemoryThresholdMB ignores memory deltas at or below this threshold.
	MemoryThresholdMB int `json:"memory_threshold_mb,omitempty" yaml:"memory_threshold_mb,omitempty"`
	// CPUThreshold ignores CPU deltas at or below this threshold.
	CPUThreshold int `json:"cpu_threshold,omitempty" yaml:"cpu_threshold,omitempty"`
	// WebhookURL receives JSON notifications for drift reports when configured.
	WebhookURL string `json:"webhook_url,omitempty" yaml:"webhook_url,omitempty"`
}

// DriftReport summarizes drift detected between a baseline snapshot and current inventory.
type DriftReport struct {
	// Baseline is the baseline snapshot metadata.
	Baseline store.SnapshotMeta `json:"baseline" yaml:"baseline"`
	// Current is metadata describing the current inventory sample.
	Current store.SnapshotMeta `json:"current" yaml:"current"`
	// Events contains every detected drift event.
	Events []DriftEvent `json:"events" yaml:"events"`
	// AddedVMs is the number of newly observed VMs.
	AddedVMs int `json:"added_vms" yaml:"added_vms"`
	// RemovedVMs is the number of missing baseline VMs.
	RemovedVMs int `json:"removed_vms" yaml:"removed_vms"`
	// ModifiedVMs is the number of VMs with field changes.
	ModifiedVMs int `json:"modified_vms" yaml:"modified_vms"`
	// PolicyDrifts is the number of policy-driven drift events.
	PolicyDrifts int `json:"policy_drifts" yaml:"policy_drifts"`
	// EvaluatedAt is when drift was evaluated.
	EvaluatedAt time.Time `json:"evaluated_at" yaml:"evaluated_at"`
}

// DriftDetector compares baseline inventory with current inventory and optional policy state.
type DriftDetector struct {
	store        store.Store
	policyEngine *PolicyEngine
	config       DriftConfig
	httpClient   *http.Client
}

// NewDriftDetector creates a drift detector backed by a state store.
func NewDriftDetector(stateStore store.Store, policyEngine *PolicyEngine, config DriftConfig) *DriftDetector {
	return &DriftDetector{
		store:        stateStore,
		policyEngine: policyEngine,
		config:       config,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Compare evaluates drift between a baseline snapshot and current inventory.
func (d *DriftDetector) Compare(ctx context.Context, baselineID string, currentResult *models.DiscoveryResult) (*DriftReport, error) {
	if d == nil {
		return nil, fmt.Errorf("compare drift: detector is nil")
	}
	if d.store == nil {
		return nil, fmt.Errorf("compare drift: store is nil")
	}
	if currentResult == nil {
		return nil, fmt.Errorf("compare drift: current result is nil")
	}

	tenantID := store.TenantIDFromContext(ctx)
	baselineResult, err := d.store.GetSnapshot(ctx, tenantID, baselineID)
	if err != nil {
		return nil, fmt.Errorf("compare drift: %w", err)
	}

	report := &DriftReport{
		Baseline: store.SnapshotMeta{
			ID:           baselineID,
			TenantID:     tenantID,
			Source:       baselineResult.Source,
			Platform:     baselineResult.Platform,
			VMCount:      len(baselineResult.VMs),
			DiscoveredAt: baselineResult.DiscoveredAt,
		},
		Current: store.SnapshotMeta{
			ID:           "current",
			TenantID:     tenantID,
			Source:       currentResult.Source,
			Platform:     currentResult.Platform,
			VMCount:      len(currentResult.VMs),
			DiscoveredAt: currentResult.DiscoveredAt,
		},
		Events:      make([]DriftEvent, 0),
		EvaluatedAt: time.Now().UTC(),
	}

	baselineVMs := indexVMs(baselineResult.VMs)
	currentVMs := indexVMs(currentResult.VMs)
	modifiedSet := make(map[string]struct{})

	for key, currentVM := range currentVMs {
		baselineVM, ok := baselineVMs[key]
		if !ok {
			report.AddedVMs++
			report.Events = append(report.Events, DriftEvent{
				Type:       DriftAdded,
				VM:         currentVM,
				Field:      "vm",
				NewValue:   currentVM.Name,
				Severity:   DriftSeverityInfo,
				DetectedAt: report.EvaluatedAt,
			})
			continue
		}

		for _, event := range d.compareVMs(baselineVM, currentVM, report.EvaluatedAt) {
			report.Events = append(report.Events, event)
			modifiedSet[key] = struct{}{}
		}
	}

	for key, baselineVM := range baselineVMs {
		if _, ok := currentVMs[key]; ok {
			continue
		}
		report.RemovedVMs++
		report.Events = append(report.Events, DriftEvent{
			Type:       DriftRemoved,
			VM:         baselineVM,
			Field:      "vm",
			OldValue:   baselineVM.Name,
			Severity:   DriftSeverityCritical,
			DetectedAt: report.EvaluatedAt,
		})
	}

	report.ModifiedVMs = len(modifiedSet)

	if d.policyEngine != nil {
		policyReport, err := d.policyEngine.Evaluate(currentResult)
		if err != nil {
			return nil, fmt.Errorf("compare drift: evaluate policy drift: %w", err)
		}
		for _, violation := range policyReport.Violations {
			report.PolicyDrifts++
			report.Events = append(report.Events, DriftEvent{
				Type:       DriftPolicyViolation,
				VM:         violation.VM,
				Field:      violation.Rule.Field,
				OldValue:   nil,
				NewValue:   violation.CurrentValue,
				Severity:   driftSeverityFromPolicy(violation.Severity),
				DetectedAt: report.EvaluatedAt,
			})
		}
	}

	return report, nil
}

// NotifyWebhook posts a drift report to the configured webhook endpoint.
func (d *DriftDetector) NotifyWebhook(report *DriftReport) error {
	if d == nil {
		return fmt.Errorf("notify webhook: detector is nil")
	}
	if report == nil {
		return fmt.Errorf("notify webhook: report is nil")
	}
	if d.config.WebhookURL == "" {
		return nil
	}

	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("notify webhook: marshal report: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, d.config.WebhookURL, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("notify webhook: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("notify webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("notify webhook: status %d", resp.StatusCode)
	}

	return nil
}

func (d *DriftDetector) compareVMs(baselineVM, currentVM models.VirtualMachine, detectedAt time.Time) []DriftEvent {
	events := make([]DriftEvent, 0)
	addEvent := func(field string, oldValue, newValue interface{}, severity DriftSeverity) {
		if d.isIgnored(field) {
			return
		}
		events = append(events, DriftEvent{
			Type:       DriftModified,
			VM:         currentVM,
			Field:      field,
			OldValue:   oldValue,
			NewValue:   newValue,
			Severity:   severity,
			DetectedAt: detectedAt,
		})
	}

	if baselineVM.CPUCount != currentVM.CPUCount && absInt(baselineVM.CPUCount-currentVM.CPUCount) > d.config.CPUThreshold {
		addEvent("cpu_count", baselineVM.CPUCount, currentVM.CPUCount, DriftSeverityWarning)
	}

	if baselineVM.MemoryMB != currentVM.MemoryMB && absInt(baselineVM.MemoryMB-currentVM.MemoryMB) > d.config.MemoryThresholdMB {
		addEvent("memory_mb", baselineVM.MemoryMB, currentVM.MemoryMB, DriftSeverityWarning)
	}

	if baselineVM.PowerState != currentVM.PowerState {
		addEvent("power_state", baselineVM.PowerState, currentVM.PowerState, DriftSeverityInfo)
	}

	if baselineVM.Host != currentVM.Host {
		addEvent("host", baselineVM.Host, currentVM.Host, DriftSeverityInfo)
	}

	if baselineVM.Cluster != currentVM.Cluster {
		addEvent("cluster", baselineVM.Cluster, currentVM.Cluster, DriftSeverityInfo)
	}

	if baselineVM.Folder != currentVM.Folder {
		addEvent("folder", baselineVM.Folder, currentVM.Folder, DriftSeverityInfo)
	}

	if !mapsEqual(baselineVM.Tags, currentVM.Tags) {
		addEvent("tags", baselineVM.Tags, currentVM.Tags, DriftSeverityInfo)
	}

	return events
}

func (d *DriftDetector) isIgnored(field string) bool {
	return slices.Contains(d.config.IgnoreFields, field)
}

func indexVMs(vms []models.VirtualMachine) map[string]models.VirtualMachine {
	items := make(map[string]models.VirtualMachine, len(vms))
	for _, vm := range vms {
		items[vmKey(vm)] = vm
	}
	return items
}

func driftSeverityFromPolicy(severity PolicySeverity) DriftSeverity {
	switch severity {
	case PolicySeverityEnforce:
		return DriftSeverityCritical
	case PolicySeverityWarn:
		return DriftSeverityWarning
	default:
		return DriftSeverityInfo
	}
}

func mapsEqual(left, right map[string]string) bool {
	if len(left) != len(right) {
		return false
	}
	for key, value := range left {
		if right[key] != value {
			return false
		}
	}
	return true
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
