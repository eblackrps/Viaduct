package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"gopkg.in/yaml.v3"
)

// MigrationSpec describes a declarative workload migration.
type MigrationSpec struct {
	Name      string             `json:"name" yaml:"name"`
	Source    SourceSpec         `json:"source" yaml:"source"`
	Target    TargetSpec         `json:"target" yaml:"target"`
	Workloads []WorkloadSelector `json:"workloads" yaml:"workloads"`
	Options   MigrationOptions   `json:"options" yaml:"options"`
}

// SourceSpec identifies the source platform and credentials reference for a migration.
type SourceSpec struct {
	Address       string          `json:"address" yaml:"address"`
	Platform      models.Platform `json:"platform" yaml:"platform"`
	CredentialRef string          `json:"credential_ref" yaml:"credential_ref"`
}

// TargetSpec identifies the target platform and defaults for a migration.
type TargetSpec struct {
	Address        string          `json:"address" yaml:"address"`
	Platform       models.Platform `json:"platform" yaml:"platform"`
	CredentialRef  string          `json:"credential_ref" yaml:"credential_ref"`
	DefaultHost    string          `json:"default_host,omitempty" yaml:"default_host,omitempty"`
	DefaultStorage string          `json:"default_storage,omitempty" yaml:"default_storage,omitempty"`
}

// WorkloadSelector chooses source VMs and applies target-specific overrides.
type WorkloadSelector struct {
	Match     MatchCriteria     `json:"match" yaml:"match"`
	Overrides WorkloadOverrides `json:"overrides,omitempty" yaml:"overrides,omitempty"`
}

// MatchCriteria defines workload selection rules.
type MatchCriteria struct {
	NamePattern string            `json:"name_pattern,omitempty" yaml:"name_pattern,omitempty"`
	Tags        map[string]string `json:"tags,omitempty" yaml:"tags,omitempty"`
	Folder      string            `json:"folder,omitempty" yaml:"folder,omitempty"`
	PowerState  models.PowerState `json:"power_state,omitempty" yaml:"power_state,omitempty"`
	Exclude     []string          `json:"exclude,omitempty" yaml:"exclude,omitempty"`
}

// WorkloadOverrides contains per-workload target placement and mapping overrides.
type WorkloadOverrides struct {
	TargetHost    string            `json:"target_host,omitempty" yaml:"target_host,omitempty"`
	TargetStorage string            `json:"target_storage,omitempty" yaml:"target_storage,omitempty"`
	NetworkMap    map[string]string `json:"network_map,omitempty" yaml:"network_map,omitempty"`
	StorageMap    map[string]string `json:"storage_map,omitempty" yaml:"storage_map,omitempty"`
	Dependencies  []string          `json:"dependencies,omitempty" yaml:"dependencies,omitempty"`
}

// ExecutionWindow constrains when a migration may execute.
type ExecutionWindow struct {
	NotBefore time.Time `json:"not_before,omitempty" yaml:"not_before,omitempty"`
	NotAfter  time.Time `json:"not_after,omitempty" yaml:"not_after,omitempty"`
}

// ApprovalGate captures approval requirements for a migration run.
type ApprovalGate struct {
	Required   bool      `json:"required,omitempty" yaml:"required,omitempty"`
	ApprovedBy string    `json:"approved_by,omitempty" yaml:"approved_by,omitempty"`
	ApprovedAt time.Time `json:"approved_at,omitempty" yaml:"approved_at,omitempty"`
	Ticket     string    `json:"ticket,omitempty" yaml:"ticket,omitempty"`
}

// Approved reports whether the approval gate has been satisfied.
func (a ApprovalGate) Approved() bool {
	return !a.Required || strings.TrimSpace(a.ApprovedBy) != ""
}

// WaveStrategy configures dependency-aware batching for larger migrations.
type WaveStrategy struct {
	Size            int  `json:"size,omitempty" yaml:"size,omitempty"`
	DependencyAware bool `json:"dependency_aware,omitempty" yaml:"dependency_aware,omitempty"`
}

// MigrationOptions contains execution-time behavior flags.
type MigrationOptions struct {
	DryRun         bool            `json:"dry_run,omitempty" yaml:"dry_run,omitempty"`
	Parallel       int             `json:"parallel,omitempty" yaml:"parallel,omitempty"`
	ShutdownSource bool            `json:"shutdown_source,omitempty" yaml:"shutdown_source,omitempty"`
	RemoveSource   bool            `json:"remove_source,omitempty" yaml:"remove_source,omitempty"`
	VerifyBoot     bool            `json:"verify_boot,omitempty" yaml:"verify_boot,omitempty"`
	Window         ExecutionWindow `json:"window,omitempty" yaml:"window,omitempty"`
	Approval       ApprovalGate    `json:"approval,omitempty" yaml:"approval,omitempty"`
	Waves          WaveStrategy    `json:"waves,omitempty" yaml:"waves,omitempty"`
}

// ParseSpec loads, normalizes, and validates a migration spec from YAML.
func ParseSpec(path string) (*MigrationSpec, error) {
	// #nosec G304 -- operators provide the spec path explicitly when loading a migration document from disk.
	payload, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("parse spec: read %s: %w", path, err)
	}

	var spec MigrationSpec
	if err := yaml.Unmarshal(payload, &spec); err != nil {
		return nil, fmt.Errorf("parse spec: decode %s: %w", path, err)
	}

	if spec.Options.Parallel <= 0 {
		spec.Options.Parallel = 1
	}
	if spec.Options.Waves.Size <= 0 {
		spec.Options.Waves.Size = spec.Options.Parallel
	}
	if spec.Name == "" {
		spec.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	if errs := ValidateSpec(&spec); len(errs) > 0 {
		messages := make([]string, 0, len(errs))
		for _, item := range errs {
			messages = append(messages, item.Error())
		}
		return nil, fmt.Errorf("parse spec: validation failed: %s", strings.Join(messages, "; "))
	}

	return &spec, nil
}

// ValidateSpec validates a migration specification and returns all detected problems.
func ValidateSpec(spec *MigrationSpec) []error {
	if spec == nil {
		return []error{fmt.Errorf("spec is nil")}
	}

	errs := make([]error, 0)
	if strings.TrimSpace(spec.Name) == "" {
		errs = append(errs, fmt.Errorf("name is required"))
	}
	if strings.TrimSpace(spec.Source.Address) == "" {
		errs = append(errs, fmt.Errorf("source.address is required"))
	}
	if !validPlatform(spec.Source.Platform) {
		errs = append(errs, fmt.Errorf("source.platform %q is invalid", spec.Source.Platform))
	}
	if strings.TrimSpace(spec.Target.Address) == "" {
		errs = append(errs, fmt.Errorf("target.address is required"))
	}
	if !validPlatform(spec.Target.Platform) {
		errs = append(errs, fmt.Errorf("target.platform %q is invalid", spec.Target.Platform))
	}
	if len(spec.Workloads) == 0 {
		errs = append(errs, fmt.Errorf("at least one workload selector is required"))
	}
	if spec.Options.Parallel < 1 {
		errs = append(errs, fmt.Errorf("options.parallel must be at least 1"))
	}
	if spec.Options.Waves.Size < 1 {
		errs = append(errs, fmt.Errorf("options.waves.size must be at least 1"))
	}
	if !spec.Options.Window.NotBefore.IsZero() && !spec.Options.Window.NotAfter.IsZero() && spec.Options.Window.NotAfter.Before(spec.Options.Window.NotBefore) {
		errs = append(errs, fmt.Errorf("options.window.not_after must be after options.window.not_before"))
	}
	if spec.Options.Approval.ApprovedAt.IsZero() != (strings.TrimSpace(spec.Options.Approval.ApprovedBy) == "") {
		errs = append(errs, fmt.Errorf("options.approval.approved_by and options.approval.approved_at must be set together"))
	}

	for idx, selector := range spec.Workloads {
		if selector.Match.NamePattern == "" && len(selector.Match.Tags) == 0 && selector.Match.Folder == "" && selector.Match.PowerState == "" {
			errs = append(errs, fmt.Errorf("workloads[%d] requires at least one match criterion", idx))
		}

		if selector.Match.PowerState != "" && !validPowerState(selector.Match.PowerState) {
			errs = append(errs, fmt.Errorf("workloads[%d].match.power_state %q is invalid", idx, selector.Match.PowerState))
		}

		if pattern := selector.Match.NamePattern; pattern != "" {
			if strings.HasPrefix(pattern, "regex:") {
				if _, err := regexp.Compile(strings.TrimPrefix(pattern, "regex:")); err != nil {
					errs = append(errs, fmt.Errorf("workloads[%d].match.name_pattern regex is invalid: %w", idx, err))
				}
			} else if _, err := filepath.Match(pattern, "sample"); err != nil {
				errs = append(errs, fmt.Errorf("workloads[%d].match.name_pattern glob is invalid: %w", idx, err))
			}
		}

		for _, excluded := range selector.Match.Exclude {
			if excluded == "" {
				errs = append(errs, fmt.Errorf("workloads[%d].match.exclude contains an empty pattern", idx))
				break
			}
		}

		for _, dependency := range selector.Overrides.Dependencies {
			if strings.TrimSpace(dependency) == "" {
				errs = append(errs, fmt.Errorf("workloads[%d].overrides.dependencies contains an empty dependency", idx))
				break
			}
		}
	}

	return errs
}

func validPlatform(platform models.Platform) bool {
	switch platform {
	case models.PlatformVMware, models.PlatformProxmox, models.PlatformHyperV, models.PlatformKVM, models.PlatformNutanix:
		return true
	default:
		return false
	}
}

func validPowerState(state models.PowerState) bool {
	switch state {
	case models.PowerOn, models.PowerOff, models.PowerSuspend, models.PowerUnknown:
		return true
	default:
		return false
	}
}
