package migrate

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/connectors"
	"github.com/eblackrps/viaduct/internal/models"
)

// CheckStatus captures the outcome of an individual pre-flight check.
type CheckStatus string

const (
	// StatusPass indicates a successful check.
	StatusPass CheckStatus = "pass"
	// StatusWarn indicates a non-blocking warning.
	StatusWarn CheckStatus = "warn"
	// StatusFail indicates a blocking failure.
	StatusFail CheckStatus = "fail"
)

// CheckResult stores the outcome of a single pre-flight validation.
type CheckResult struct {
	Name     string        `json:"name"`
	Status   CheckStatus   `json:"status"`
	Message  string        `json:"message"`
	Duration time.Duration `json:"duration"`
}

// PreflightReport aggregates all pre-flight check results for a migration.
type PreflightReport struct {
	// Plan is the derived execution plan when planning succeeded.
	Plan *MigrationPlan `json:"plan,omitempty"`
	// Checks contains the individual preflight results.
	Checks []CheckResult `json:"checks"`
	// PassCount is the number of passing checks.
	PassCount int `json:"pass_count"`
	// WarnCount is the number of warning checks.
	WarnCount int `json:"warn_count"`
	// FailCount is the number of failed checks.
	FailCount int `json:"fail_count"`
	// CanProceed reports whether blocking failures were found.
	CanProceed bool `json:"can_proceed"`
}

// PreflightChecker validates migration prerequisites against source and target systems.
type PreflightChecker struct {
	source       connectors.Connector
	target       connectors.Connector
	spec         *MigrationSpec
	sourceResult *models.DiscoveryResult
	targetResult *models.DiscoveryResult
	plan         *MigrationPlan
}

// NewPreflightChecker creates a new pre-flight checker.
func NewPreflightChecker(source, target connectors.Connector, spec *MigrationSpec) *PreflightChecker {
	return &PreflightChecker{
		source: source,
		target: target,
		spec:   spec,
	}
}

// RunAll executes every pre-flight check and returns the full report.
func (c *PreflightChecker) RunAll(ctx context.Context) (*PreflightReport, error) {
	if c.spec == nil {
		return nil, fmt.Errorf("run preflight: spec is nil")
	}

	checks := []func(context.Context) CheckResult{
		c.checkSourceConnectivity,
		c.checkTargetConnectivity,
		c.checkExecutionWindow,
		c.checkApprovalGate,
		c.checkDiskSpace,
		c.checkNetworkMappings,
		c.checkNameConflicts,
		c.checkSourceBackup,
		c.checkDiskFormats,
		c.checkResourceAvailability,
		c.checkRollbackReadiness,
		c.checkExecutionPlan,
	}

	report := &PreflightReport{
		Checks: make([]CheckResult, 0, len(checks)),
	}

	for _, check := range checks {
		result := check(ctx)
		report.Checks = append(report.Checks, result)
		switch result.Status {
		case StatusPass:
			report.PassCount++
		case StatusWarn:
			report.WarnCount++
		case StatusFail:
			report.FailCount++
		}
	}

	report.CanProceed = report.FailCount == 0
	report.Plan = c.plan
	return report, nil
}

func (c *PreflightChecker) checkExecutionWindow(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.spec == nil {
		return CheckResult{Name: "execution-window", Status: StatusFail, Message: "migration specification is unavailable", Duration: time.Since(startedAt)}
	}

	window := c.spec.Options.Window
	now := time.Now().UTC()
	if !window.NotBefore.IsZero() && now.Before(window.NotBefore) {
		return CheckResult{Name: "execution-window", Status: StatusFail, Message: fmt.Sprintf("migration window opens at %s", window.NotBefore.Format(time.RFC3339)), Duration: time.Since(startedAt)}
	}
	if !window.NotAfter.IsZero() && now.After(window.NotAfter) {
		return CheckResult{Name: "execution-window", Status: StatusFail, Message: fmt.Sprintf("migration window closed at %s", window.NotAfter.Format(time.RFC3339)), Duration: time.Since(startedAt)}
	}

	return CheckResult{Name: "execution-window", Status: StatusPass, Message: "execution window allows this migration run", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkApprovalGate(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.spec == nil {
		return CheckResult{Name: "approval-gate", Status: StatusFail, Message: "migration specification is unavailable", Duration: time.Since(startedAt)}
	}
	if c.spec.Options.Approval.Required && !c.spec.Options.Approval.Approved() {
		return CheckResult{Name: "approval-gate", Status: StatusFail, Message: "migration requires approval before execution", Duration: time.Since(startedAt)}
	}

	return CheckResult{Name: "approval-gate", Status: StatusPass, Message: "approval gate is satisfied", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkSourceConnectivity(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if err := c.source.Connect(ctx); err != nil {
		return CheckResult{Name: "source-connectivity", Status: StatusFail, Message: fmt.Sprintf("source connection failed: %v", err), Duration: time.Since(startedAt)}
	}
	// Connector shutdown is best effort for preflight probes.
	defer func() { _ = c.source.Close() }()

	result, err := c.source.Discover(ctx)
	if err != nil {
		return CheckResult{Name: "source-connectivity", Status: StatusFail, Message: fmt.Sprintf("source discovery failed: %v", err), Duration: time.Since(startedAt)}
	}

	c.sourceResult = result
	return CheckResult{Name: "source-connectivity", Status: StatusPass, Message: "source platform is reachable", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkTargetConnectivity(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if err := c.target.Connect(ctx); err != nil {
		return CheckResult{Name: "target-connectivity", Status: StatusFail, Message: fmt.Sprintf("target connection failed: %v", err), Duration: time.Since(startedAt)}
	}
	// Connector shutdown is best effort for preflight probes.
	defer func() { _ = c.target.Close() }()

	result, err := c.target.Discover(ctx)
	if err != nil {
		return CheckResult{Name: "target-connectivity", Status: StatusFail, Message: fmt.Sprintf("target discovery failed: %v", err), Duration: time.Since(startedAt)}
	}

	c.targetResult = result
	return CheckResult{Name: "target-connectivity", Status: StatusPass, Message: "target platform is reachable", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkDiskSpace(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil || c.targetResult == nil {
		return CheckResult{Name: "disk-space", Status: StatusWarn, Message: "connectivity checks did not complete, disk space could not be verified", Duration: time.Since(startedAt)}
	}

	selected := MatchWorkloads(c.sourceResult.VMs, c.spec.Workloads)
	var requiredMB int64
	for _, vm := range selected {
		for _, disk := range vm.Disks {
			requiredMB += int64(disk.SizeMB)
		}
	}

	var freeMB int64
	for _, datastore := range c.targetResult.Datastores {
		freeMB += datastore.FreeMB
	}

	if freeMB == 0 {
		return CheckResult{Name: "disk-space", Status: StatusWarn, Message: "target free space is unknown", Duration: time.Since(startedAt)}
	}
	if requiredMB > freeMB {
		return CheckResult{Name: "disk-space", Status: StatusFail, Message: fmt.Sprintf("target free space %d MB is smaller than required %d MB", freeMB, requiredMB), Duration: time.Since(startedAt)}
	}

	return CheckResult{Name: "disk-space", Status: StatusPass, Message: fmt.Sprintf("target free space %d MB is sufficient for %d MB", freeMB, requiredMB), Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkNetworkMappings(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil || c.targetResult == nil {
		return CheckResult{Name: "network-mappings", Status: StatusWarn, Message: "connectivity checks did not complete, network mappings could not be verified", Duration: time.Since(startedAt)}
	}

	workloads := buildWorkloadMigrations(c.sourceResult.VMs, c.spec.Workloads)
	errs := make([]error, 0)
	for _, workload := range workloads {
		mapper := NewNetworkMapper(workload.NetworkMappings, c.targetResult.Networks)
		errs = append(errs, mapper.ValidateTargetNetworks()...)
		_, nicErrors := mapper.MapAllNICs(workload.VM.NICs)
		errs = append(errs, nicErrors...)
	}

	if len(errs) > 0 {
		return CheckResult{Name: "network-mappings", Status: StatusFail, Message: errs[0].Error(), Duration: time.Since(startedAt)}
	}

	return CheckResult{Name: "network-mappings", Status: StatusPass, Message: "all target networks are available", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkNameConflicts(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil || c.targetResult == nil {
		return CheckResult{Name: "name-conflicts", Status: StatusWarn, Message: "connectivity checks did not complete, name conflicts could not be verified", Duration: time.Since(startedAt)}
	}

	selected := MatchWorkloads(c.sourceResult.VMs, c.spec.Workloads)
	targetNames := make(map[string]struct{}, len(c.targetResult.VMs))
	for _, vm := range c.targetResult.VMs {
		targetNames[strings.ToLower(vm.Name)] = struct{}{}
	}

	for _, vm := range selected {
		if _, ok := targetNames[strings.ToLower(vm.Name)]; ok {
			return CheckResult{Name: "name-conflicts", Status: StatusFail, Message: fmt.Sprintf("target already contains VM %q", vm.Name), Duration: time.Since(startedAt)}
		}
	}

	return CheckResult{Name: "name-conflicts", Status: StatusPass, Message: "no target naming conflicts detected", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkSourceBackup(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil {
		return CheckResult{Name: "source-backup", Status: StatusWarn, Message: "source inventory unavailable, backup presence could not be verified", Duration: time.Since(startedAt)}
	}

	selected := MatchWorkloads(c.sourceResult.VMs, c.spec.Workloads)
	for _, vm := range selected {
		if len(vm.Snapshots) > 0 {
			return CheckResult{Name: "source-backup", Status: StatusPass, Message: "source recovery points detected", Duration: time.Since(startedAt)}
		}
	}

	return CheckResult{Name: "source-backup", Status: StatusWarn, Message: "no source snapshots or backups detected", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkDiskFormats(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil {
		return CheckResult{Name: "disk-formats", Status: StatusWarn, Message: "source inventory unavailable, disk formats could not be verified", Duration: time.Since(startedAt)}
	}

	selected := MatchWorkloads(c.sourceResult.VMs, c.spec.Workloads)
	targetFormat := targetDiskFormat(c.spec.Target.Platform)
	for _, vm := range selected {
		for _, disk := range vm.Disks {
			sourceFormat := inferDiskFormat(disk.Name, vm.Platform)
			if _, err := qemuFormat(sourceFormat); err != nil {
				return CheckResult{Name: "disk-formats", Status: StatusFail, Message: fmt.Sprintf("VM %s has unsupported source format %q", vm.Name, sourceFormat), Duration: time.Since(startedAt)}
			}
			if _, err := qemuFormat(targetFormat); err != nil {
				return CheckResult{Name: "disk-formats", Status: StatusFail, Message: fmt.Sprintf("target format %q is unsupported", targetFormat), Duration: time.Since(startedAt)}
			}
		}
	}

	return CheckResult{Name: "disk-formats", Status: StatusPass, Message: "source disks can be converted for the target platform", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkResourceAvailability(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil || c.targetResult == nil {
		return CheckResult{Name: "resource-availability", Status: StatusWarn, Message: "connectivity checks did not complete, target capacity could not be verified", Duration: time.Since(startedAt)}
	}

	selected := MatchWorkloads(c.sourceResult.VMs, c.spec.Workloads)
	var requiredCPU int
	var requiredMemoryMB int64
	for _, vm := range selected {
		requiredCPU += vm.CPUCount
		requiredMemoryMB += int64(vm.MemoryMB)
	}

	var targetCPU int
	var targetMemoryMB int64
	for _, host := range c.targetResult.Hosts {
		targetCPU += host.CPUCores
		targetMemoryMB += host.MemoryMB
	}
	if targetCPU == 0 || targetMemoryMB == 0 {
		return CheckResult{Name: "resource-availability", Status: StatusWarn, Message: "target capacity is unknown", Duration: time.Since(startedAt)}
	}

	if requiredCPU > targetCPU || requiredMemoryMB > targetMemoryMB {
		return CheckResult{
			Name:     "resource-availability",
			Status:   StatusFail,
			Message:  fmt.Sprintf("target capacity is insufficient for %d CPU / %d MB", requiredCPU, requiredMemoryMB),
			Duration: time.Since(startedAt),
		}
	}

	return CheckResult{
		Name:     "resource-availability",
		Status:   StatusPass,
		Message:  fmt.Sprintf("target capacity can host %d CPU / %d MB", requiredCPU, requiredMemoryMB),
		Duration: time.Since(startedAt),
	}
}

func (c *PreflightChecker) checkRollbackReadiness(ctx context.Context) CheckResult {
	startedAt := time.Now()

	missing := make([]string, 0)
	if _, ok := c.source.(vmSnapshotter); !ok {
		missing = append(missing, "source snapshot support")
	}
	if _, ok := c.source.(vmPowerController); !ok {
		missing = append(missing, "source power control")
	}
	if _, ok := c.target.(vmRemover); !ok {
		missing = append(missing, "target VM removal")
	}

	if len(missing) > 0 {
		return CheckResult{
			Name:     "rollback-readiness",
			Status:   StatusWarn,
			Message:  fmt.Sprintf("rollback will be limited because %s are unavailable", strings.Join(missing, ", ")),
			Duration: time.Since(startedAt),
		}
	}

	return CheckResult{Name: "rollback-readiness", Status: StatusPass, Message: "rollback dependencies are available", Duration: time.Since(startedAt)}
}

func (c *PreflightChecker) checkExecutionPlan(ctx context.Context) CheckResult {
	startedAt := time.Now()
	if c.sourceResult == nil {
		return CheckResult{Name: "execution-plan", Status: StatusWarn, Message: "source inventory unavailable, execution plan could not be derived", Duration: time.Since(startedAt)}
	}

	plan, err := BuildExecutionPlan(c.spec, c.sourceResult.VMs)
	if err != nil {
		return CheckResult{Name: "execution-plan", Status: StatusFail, Message: err.Error(), Duration: time.Since(startedAt)}
	}
	c.plan = plan

	return CheckResult{
		Name:     "execution-plan",
		Status:   StatusPass,
		Message:  fmt.Sprintf("planned %d workloads across %d migration wave(s)", plan.TotalWorkloads, len(plan.Waves)),
		Duration: time.Since(startedAt),
	}
}
