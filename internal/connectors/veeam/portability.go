package veeam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

// BackupJobTemplate describes a portable backup job configuration for recreation.
type BackupJobTemplate struct {
	// Name is the target backup job name.
	Name string `json:"name" yaml:"name"`
	// Type is the backup job type.
	Type string `json:"type" yaml:"type"`
	// Schedule is the preserved backup schedule.
	Schedule string `json:"schedule" yaml:"schedule"`
	// TargetRepo is the mapped target repository.
	TargetRepo string `json:"target_repo" yaml:"target_repo"`
	// RetentionDays is the preserved retention value.
	RetentionDays int `json:"retention_days" yaml:"retention_days"`
	// ProtectedVMs contains the target VM names protected by the job.
	ProtectedVMs []string `json:"protected_vms" yaml:"protected_vms"`
	// Enabled reports whether the job should be created enabled.
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// JobMigrationPlan describes how backup jobs will move with a workload migration.
type JobMigrationPlan struct {
	// SourceVM is the source workload name.
	SourceVM models.VirtualMachine `json:"source_vm" yaml:"source_vm"`
	// TargetVM is the target workload name.
	TargetVM models.VirtualMachine `json:"target_vm" yaml:"target_vm"`
	// Jobs are the portable backup job templates to recreate.
	Jobs []BackupJobTemplate `json:"jobs" yaml:"jobs"`
	// RepositoryMappings remaps source repositories to target repositories.
	RepositoryMappings map[string]string `json:"repository_mappings,omitempty" yaml:"repository_mappings,omitempty"`
	// Warnings contains non-fatal portability warnings.
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
	// RepositoryCompatibility captures source-to-target repository compatibility findings.
	RepositoryCompatibility []RepositoryCompatibility `json:"repository_compatibility,omitempty" yaml:"repository_compatibility,omitempty"`
}

// JobMigrationResult records created jobs and their verification state.
type JobMigrationResult struct {
	// CreatedJobs are the identifiers of jobs created on the target side.
	CreatedJobs []string `json:"created_jobs" yaml:"created_jobs"`
	// VerificationStatus records the verification result per created job.
	VerificationStatus map[string]string `json:"verification_status" yaml:"verification_status"`
	// Errors contains any non-fatal job migration errors.
	Errors []string `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// BackupPolicyDrift captures a mismatch between the expected and observed backup policy after migration.
type BackupPolicyDrift struct {
	// JobName is the recreated backup job with the observed drift.
	JobName string `json:"job_name" yaml:"job_name"`
	// Field is the job setting that drifted.
	Field string `json:"field" yaml:"field"`
	// Expected is the desired value recorded in the portability plan.
	Expected string `json:"expected" yaml:"expected"`
	// Actual is the observed value returned by the backup platform.
	Actual string `json:"actual" yaml:"actual"`
	// Severity is a concise indicator for operators and reports.
	Severity string `json:"severity" yaml:"severity"`
}

// BackupContinuityReport summarizes post-migration backup continuity for one workload.
type BackupContinuityReport struct {
	// TargetVM is the migrated workload being validated.
	TargetVM models.VirtualMachine `json:"target_vm" yaml:"target_vm"`
	// Status is the overall continuity state for the migrated workload.
	Status string `json:"status" yaml:"status"`
	// JobsValidated is the number of planned jobs evaluated against the target platform.
	JobsValidated int `json:"jobs_validated" yaml:"jobs_validated"`
	// RestorePointCount is the number of restore points currently protecting the target VM.
	RestorePointCount int `json:"restore_point_count" yaml:"restore_point_count"`
	// LatestRestorePointAt is the newest restore point timestamp when present.
	LatestRestorePointAt time.Time `json:"latest_restore_point_at,omitempty" yaml:"latest_restore_point_at,omitempty"`
	// RepositoryCompatibility contains repository validation details for the recreated jobs.
	RepositoryCompatibility []RepositoryCompatibility `json:"repository_compatibility,omitempty" yaml:"repository_compatibility,omitempty"`
	// PolicyDrifts contains detected post-migration drift in recreated jobs.
	PolicyDrifts []BackupPolicyDrift `json:"policy_drifts,omitempty" yaml:"policy_drifts,omitempty"`
	// VerificationStatus records per-job verification state from execution when available.
	VerificationStatus map[string]string `json:"verification_status,omitempty" yaml:"verification_status,omitempty"`
	// Warnings contains actionable continuity warnings.
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// FleetBackupContinuityReport aggregates post-migration backup continuity across multiple workloads.
type FleetBackupContinuityReport struct {
	// Reports contains one continuity report per migrated target workload.
	Reports map[string]BackupContinuityReport `json:"reports" yaml:"reports"`
	// Warnings contains fleet-level warnings gathered during validation.
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// RepositoryCompatibility summarizes compatibility findings for a repository mapping.
type RepositoryCompatibility struct {
	// SourceRepository is the repository used by the source-side backup job.
	SourceRepository string `json:"source_repository" yaml:"source_repository"`
	// TargetRepository is the repository selected on the target side.
	TargetRepository string `json:"target_repository" yaml:"target_repository"`
	// Compatible indicates whether the target repository can accept recreated jobs.
	Compatible bool `json:"compatible" yaml:"compatible"`
	// Reason provides a concise explanation when compatibility is degraded or failed.
	Reason string `json:"reason,omitempty" yaml:"reason,omitempty"`
}

// WorkloadMapping describes a source-to-target workload pair for portability planning.
type WorkloadMapping struct {
	// SourceVM is the source workload protected by the original backup job.
	SourceVM models.VirtualMachine `json:"source_vm" yaml:"source_vm"`
	// TargetVM is the migrated workload that should inherit backup protection.
	TargetVM models.VirtualMachine `json:"target_vm" yaml:"target_vm"`
}

// FleetJobMigrationPlan aggregates portability plans across multiple workload migrations.
type FleetJobMigrationPlan struct {
	// Plans contains one portability plan per workload mapping.
	Plans []JobMigrationPlan `json:"plans" yaml:"plans"`
	// Warnings contains non-fatal planning warnings across the fleet.
	Warnings []string `json:"warnings,omitempty" yaml:"warnings,omitempty"`
}

// FleetJobMigrationResult aggregates backup portability execution across multiple workloads.
type FleetJobMigrationResult struct {
	// Results contains per-workload execution results keyed by target VM name.
	Results map[string]JobMigrationResult `json:"results" yaml:"results"`
	// Errors contains any fleet-level execution errors.
	Errors []string `json:"errors,omitempty" yaml:"errors,omitempty"`
}

// PortabilityManager manages Veeam backup job migration across workload moves.
type PortabilityManager struct {
	client *VeeamClient
}

// NewPortabilityManager creates a backup job portability manager.
func NewPortabilityManager(client *VeeamClient) *PortabilityManager {
	return &PortabilityManager{client: client}
}

// PlanJobMigration builds a portable backup job plan for a migrating workload.
func (m *PortabilityManager) PlanJobMigration(ctx context.Context, sourceVM, targetVM models.VirtualMachine, repositoryMappings map[string]string) (*JobMigrationPlan, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("plan job migration: client is nil")
	}

	jobPayload, err := m.client.Get(ctx, "/v1/jobs")
	if err != nil {
		return nil, fmt.Errorf("plan job migration: list jobs: %w", err)
	}
	repositoryPayload, err := m.client.Get(ctx, "/v1/backupInfrastructure/repositories")
	if err != nil {
		return nil, fmt.Errorf("plan job migration: list repositories: %w", err)
	}

	jobItems, err := decodeObjectSlice(jobPayload)
	if err != nil {
		return nil, fmt.Errorf("plan job migration: decode jobs: %w", err)
	}
	repositoryItems, err := decodeObjectSlice(repositoryPayload)
	if err != nil {
		return nil, fmt.Errorf("plan job migration: decode repositories: %w", err)
	}

	repositories := make(map[string]models.BackupRepository, len(repositoryItems))
	for _, item := range repositoryItems {
		repo := mapRepository(item)
		repositories[strings.ToLower(repo.Name)] = repo
	}

	plan := &JobMigrationPlan{
		SourceVM:                sourceVM,
		TargetVM:                targetVM,
		Jobs:                    make([]BackupJobTemplate, 0),
		RepositoryMappings:      cloneStringMap(repositoryMappings),
		Warnings:                make([]string, 0),
		RepositoryCompatibility: make([]RepositoryCompatibility, 0),
	}

	for _, item := range jobItems {
		job := mapJob(item)
		if !jobProtectsVM(job, sourceVM) {
			continue
		}

		targetRepository := resolveRepository(job.TargetRepo, repositoryMappings)
		compatibility := evaluateRepositoryCompatibility(job.TargetRepo, targetRepository, repositories)
		if !compatibility.Compatible {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("repository %q is not present on the target side", targetRepository))
		}
		plan.RepositoryCompatibility = append(plan.RepositoryCompatibility, compatibility)

		plan.Jobs = append(plan.Jobs, BackupJobTemplate{
			Name:          rewriteJobName(job.Name, sourceVM.Name, targetVM.Name),
			Type:          job.Type,
			Schedule:      job.Schedule,
			TargetRepo:    targetRepository,
			RetentionDays: job.RetentionDays,
			ProtectedVMs:  []string{targetVM.Name},
			Enabled:       job.Enabled,
		})
	}

	return plan, nil
}

// PlanFleetMigration builds portable backup job plans for multiple workload migrations.
func (m *PortabilityManager) PlanFleetMigration(ctx context.Context, mappings []WorkloadMapping, repositoryMappings map[string]string) (*FleetJobMigrationPlan, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("plan fleet migration: client is nil")
	}

	plan := &FleetJobMigrationPlan{
		Plans:    make([]JobMigrationPlan, 0, len(mappings)),
		Warnings: make([]string, 0),
	}
	for _, mapping := range mappings {
		workloadPlan, err := m.PlanJobMigration(ctx, mapping.SourceVM, mapping.TargetVM, repositoryMappings)
		if err != nil {
			return nil, fmt.Errorf("plan fleet migration for %s: %w", mapping.TargetVM.Name, err)
		}
		plan.Plans = append(plan.Plans, *workloadPlan)
		plan.Warnings = append(plan.Warnings, workloadPlan.Warnings...)
	}

	return plan, nil
}

// ExecuteJobMigration creates portable backup jobs and validates them.
func (m *PortabilityManager) ExecuteJobMigration(ctx context.Context, plan *JobMigrationPlan) (*JobMigrationResult, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("execute job migration: client is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("execute job migration: plan is nil")
	}

	result := &JobMigrationResult{
		CreatedJobs:        make([]string, 0, len(plan.Jobs)),
		VerificationStatus: make(map[string]string, len(plan.Jobs)),
		Errors:             make([]string, 0),
	}

	for _, job := range plan.Jobs {
		response, err := m.client.Post(ctx, "/v1/jobs", job)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			log.Printf("component=veeam-portability job=%s target_repo=%s message=%q", job.Name, job.TargetRepo, fmt.Sprintf("create job failed: %v", err))
			continue
		}

		createdID := parseCreatedJobID(response, job.Name)
		result.CreatedJobs = append(result.CreatedJobs, createdID)
		log.Printf("component=veeam-portability job=%s job_id=%s message=%q", job.Name, createdID, "created portable backup job")

		if _, err := m.client.Post(ctx, "/v1/jobs/"+createdID+"/start", map[string]string{"mode": "verification"}); err != nil {
			result.VerificationStatus[createdID] = "failed"
			result.Errors = append(result.Errors, err.Error())
			log.Printf("component=veeam-portability job=%s job_id=%s message=%q", job.Name, createdID, fmt.Sprintf("verification failed: %v", err))
			continue
		}
		result.VerificationStatus[createdID] = "verified"
		log.Printf("component=veeam-portability job=%s job_id=%s message=%q", job.Name, createdID, "verification succeeded")
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("execute job migration: completed with %d error(s): %s", len(result.Errors), strings.Join(result.Errors, "; "))
	}

	return result, nil
}

// ExecuteFleetMigration recreates backup jobs for multiple migrated workloads.
func (m *PortabilityManager) ExecuteFleetMigration(ctx context.Context, plan *FleetJobMigrationPlan) (*FleetJobMigrationResult, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("execute fleet migration: client is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("execute fleet migration: plan is nil")
	}

	result := &FleetJobMigrationResult{
		Results: make(map[string]JobMigrationResult, len(plan.Plans)),
		Errors:  make([]string, 0),
	}

	for _, workloadPlan := range plan.Plans {
		workloadResult, err := m.ExecuteJobMigration(ctx, &workloadPlan)
		key := workloadPlan.TargetVM.Name
		if strings.TrimSpace(key) == "" {
			key = workloadPlan.SourceVM.Name
		}
		if workloadResult != nil {
			result.Results[key] = *workloadResult
		}
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		}
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("execute fleet migration: completed with %d error(s): %s", len(result.Errors), strings.Join(result.Errors, "; "))
	}
	return result, nil
}

// ValidateBackupContinuity verifies that recreated jobs and restore points match the expected post-migration policy.
func (m *PortabilityManager) ValidateBackupContinuity(ctx context.Context, plan *JobMigrationPlan, result *JobMigrationResult) (*BackupContinuityReport, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("validate backup continuity: client is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("validate backup continuity: plan is nil")
	}

	jobPayload, err := m.client.Get(ctx, "/v1/jobs")
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: list jobs: %w", err)
	}
	restorePayload, err := m.client.Get(ctx, "/v1/objectRestorePoints")
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: list restore points: %w", err)
	}
	repositoryPayload, err := m.client.Get(ctx, "/v1/backupInfrastructure/repositories")
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: list repositories: %w", err)
	}

	jobItems, err := decodeObjectSlice(jobPayload)
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: decode jobs: %w", err)
	}
	restoreItems, err := decodeObjectSlice(restorePayload)
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: decode restore points: %w", err)
	}
	repositoryItems, err := decodeObjectSlice(repositoryPayload)
	if err != nil {
		return nil, fmt.Errorf("validate backup continuity: decode repositories: %w", err)
	}

	currentJobs := make(map[string]models.BackupJob, len(jobItems))
	for _, item := range jobItems {
		job := mapJob(item)
		currentJobs[strings.ToLower(strings.TrimSpace(job.Name))] = job
	}
	repositories := make(map[string]models.BackupRepository, len(repositoryItems))
	for _, item := range repositoryItems {
		repo := mapRepository(item)
		repositories[strings.ToLower(strings.TrimSpace(repo.Name))] = repo
	}

	report := &BackupContinuityReport{
		TargetVM:                plan.TargetVM,
		Status:                  "healthy",
		JobsValidated:           len(plan.Jobs),
		RepositoryCompatibility: make([]RepositoryCompatibility, 0, len(plan.Jobs)),
		PolicyDrifts:            make([]BackupPolicyDrift, 0),
		VerificationStatus:      make(map[string]string),
		Warnings:                append([]string(nil), plan.Warnings...),
	}
	if result != nil {
		for key, value := range result.VerificationStatus {
			report.VerificationStatus[key] = value
		}
	}

	for _, job := range plan.Jobs {
		actual, ok := currentJobs[strings.ToLower(strings.TrimSpace(job.Name))]
		if !ok {
			report.PolicyDrifts = append(report.PolicyDrifts, BackupPolicyDrift{
				JobName:  job.Name,
				Field:    "presence",
				Expected: "present",
				Actual:   "missing",
				Severity: "error",
			})
			report.Warnings = append(report.Warnings, fmt.Sprintf("backup job %q is missing on the target side", job.Name))
			report.Status = "degraded"
			continue
		}

		report.RepositoryCompatibility = append(report.RepositoryCompatibility, evaluateRepositoryCompatibility(job.TargetRepo, actual.TargetRepo, repositories))
		report.PolicyDrifts = append(report.PolicyDrifts, detectPolicyDrift(job, actual)...)
	}

	restorePoints := matchingRestorePoints(restoreItems, plan.TargetVM, plan.Jobs)
	report.RestorePointCount = len(restorePoints)
	report.LatestRestorePointAt = latestRestorePointAt(restorePoints)
	if report.RestorePointCount == 0 {
		report.Warnings = append(report.Warnings, fmt.Sprintf("no restore points found yet for %q", plan.TargetVM.Name))
		if report.Status == "healthy" {
			report.Status = "warning"
		}
	}
	if report.Status == "healthy" && len(report.PolicyDrifts) > 0 {
		report.Status = "warning"
	}
	for _, compatibility := range report.RepositoryCompatibility {
		if !compatibility.Compatible {
			if report.Status == "healthy" {
				report.Status = "warning"
			}
			report.Warnings = append(report.Warnings, fmt.Sprintf("repository compatibility warning for %q: %s", compatibility.TargetRepository, compatibility.Reason))
		}
	}
	for _, drift := range report.PolicyDrifts {
		if drift.Severity == "error" {
			report.Status = "degraded"
			break
		}
	}

	return report, nil
}

// ValidateFleetContinuity verifies backup continuity for multiple workload migrations.
func (m *PortabilityManager) ValidateFleetContinuity(ctx context.Context, plan *FleetJobMigrationPlan, result *FleetJobMigrationResult) (*FleetBackupContinuityReport, error) {
	if m == nil || m.client == nil {
		return nil, fmt.Errorf("validate fleet continuity: client is nil")
	}
	if plan == nil {
		return nil, fmt.Errorf("validate fleet continuity: plan is nil")
	}

	report := &FleetBackupContinuityReport{
		Reports:  make(map[string]BackupContinuityReport, len(plan.Plans)),
		Warnings: append([]string(nil), plan.Warnings...),
	}

	for _, workloadPlan := range plan.Plans {
		key := workloadPlan.TargetVM.Name
		workloadResult := (*JobMigrationResult)(nil)
		if result != nil {
			if current, ok := result.Results[key]; ok {
				copy := current
				workloadResult = &copy
			}
		}

		continuity, err := m.ValidateBackupContinuity(ctx, &workloadPlan, workloadResult)
		if err != nil {
			return nil, fmt.Errorf("validate fleet continuity for %s: %w", key, err)
		}
		report.Reports[key] = *continuity
		report.Warnings = append(report.Warnings, continuity.Warnings...)
	}

	return report, nil
}

// RollbackJobMigration removes created backup jobs after a rollback.
func (m *PortabilityManager) RollbackJobMigration(ctx context.Context, result *JobMigrationResult) error {
	if m == nil || m.client == nil {
		return fmt.Errorf("rollback job migration: client is nil")
	}
	if result == nil {
		return fmt.Errorf("rollback job migration: result is nil")
	}

	var errors []string
	for _, jobID := range result.CreatedJobs {
		if err := m.client.Delete(ctx, "/v1/jobs/"+jobID); err != nil {
			errors = append(errors, err.Error())
			log.Printf("component=veeam-portability job_id=%s message=%q", jobID, fmt.Sprintf("rollback delete failed: %v", err))
			continue
		}
		log.Printf("component=veeam-portability job_id=%s message=%q", jobID, "rolled back portable backup job")
	}

	if len(errors) > 0 {
		return fmt.Errorf("rollback job migration: %s", strings.Join(errors, "; "))
	}
	return nil
}

func jobProtectsVM(job models.BackupJob, vm models.VirtualMachine) bool {
	for _, protectedVM := range job.ProtectedVMs {
		if strings.EqualFold(protectedVM, vm.Name) || strings.EqualFold(protectedVM, vm.ID) {
			return true
		}
	}
	return false
}

func resolveRepository(sourceRepository string, repositoryMappings map[string]string) string {
	if mapped, ok := repositoryMappings[sourceRepository]; ok && mapped != "" {
		return mapped
	}
	return sourceRepository
}

func rewriteJobName(name, sourceVMName, targetVMName string) string {
	if sourceVMName == "" || targetVMName == "" {
		return name
	}
	replaced := strings.ReplaceAll(name, sourceVMName, targetVMName)
	if replaced == name {
		return name + " - " + targetVMName
	}
	return replaced
}

func detectPolicyDrift(expected BackupJobTemplate, actual models.BackupJob) []BackupPolicyDrift {
	drifts := make([]BackupPolicyDrift, 0)
	appendDrift := func(field, expectedValue, actualValue, severity string) {
		if expectedValue == actualValue {
			return
		}
		drifts = append(drifts, BackupPolicyDrift{
			JobName:  expected.Name,
			Field:    field,
			Expected: expectedValue,
			Actual:   actualValue,
			Severity: severity,
		})
	}

	appendDrift("schedule", expected.Schedule, actual.Schedule, "warning")
	appendDrift("target_repo", expected.TargetRepo, actual.TargetRepo, "error")
	appendDrift("retention_days", fmt.Sprintf("%d", expected.RetentionDays), fmt.Sprintf("%d", actual.RetentionDays), "warning")
	appendDrift("enabled", fmt.Sprintf("%t", expected.Enabled), fmt.Sprintf("%t", actual.Enabled), "warning")
	appendDrift("protected_vms", strings.Join(expected.ProtectedVMs, ","), strings.Join(actual.ProtectedVMs, ","), "error")

	return drifts
}

func matchingRestorePoints(items []map[string]interface{}, targetVM models.VirtualMachine, jobs []BackupJobTemplate) []models.RestorePoint {
	jobNames := make(map[string]struct{}, len(jobs))
	for _, job := range jobs {
		jobNames[strings.ToLower(strings.TrimSpace(job.Name))] = struct{}{}
	}

	points := make([]models.RestorePoint, 0)
	for _, item := range items {
		point := mapRestorePoint(item)
		if strings.EqualFold(point.VMName, targetVM.Name) || strings.EqualFold(point.VMID, targetVM.ID) {
			points = append(points, point)
			continue
		}
		if _, ok := jobNames[strings.ToLower(strings.TrimSpace(point.JobName))]; ok {
			points = append(points, point)
		}
	}
	return points
}

func latestRestorePointAt(points []models.RestorePoint) time.Time {
	latest := time.Time{}
	for _, point := range points {
		if point.CreatedAt.After(latest) {
			latest = point.CreatedAt
		}
	}
	return latest
}

func parseCreatedJobID(payload []byte, fallback string) string {
	var response struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(payload, &response); err == nil && response.ID != "" {
		return response.ID
	}
	return fallback
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return nil
	}

	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func evaluateRepositoryCompatibility(sourceRepository, targetRepository string, repositories map[string]models.BackupRepository) RepositoryCompatibility {
	report := RepositoryCompatibility{
		SourceRepository: sourceRepository,
		TargetRepository: targetRepository,
		Compatible:       true,
	}

	target, ok := repositories[strings.ToLower(targetRepository)]
	if !ok {
		report.Compatible = false
		report.Reason = "target repository not found"
		return report
	}

	if target.FreeMB <= 0 {
		report.Compatible = false
		report.Reason = "target repository has no reported free capacity"
		return report
	}

	if target.Type == "" {
		report.Reason = "target repository type not reported"
	}

	return report
}
