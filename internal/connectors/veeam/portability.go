package veeam

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

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
