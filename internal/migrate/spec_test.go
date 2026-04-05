package migrate

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestParseSpec_ValidFull(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec(filepath.Join("..", "..", "configs", "example-migration.yaml"))
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	if spec.Name != "vmware-to-proxmox-phase2" {
		t.Fatalf("Name = %q, want vmware-to-proxmox-phase2", spec.Name)
	}
	if spec.Target.DefaultStorage != "local-lvm" {
		t.Fatalf("DefaultStorage = %q, want local-lvm", spec.Target.DefaultStorage)
	}
	if spec.Options.Parallel != 2 {
		t.Fatalf("Parallel = %d, want 2", spec.Options.Parallel)
	}
}

func TestParseSpec_ValidMinimal(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec(filepath.Join("..", "..", "configs", "example-migration-minimal.yaml"))
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	if len(spec.Workloads) != 1 {
		t.Fatalf("len(Workloads) = %d, want 1", len(spec.Workloads))
	}
}

func TestParseSpec_LabMigrationWindow_Parses(t *testing.T) {
	t.Parallel()

	spec, err := ParseSpec(filepath.Join("..", "..", "examples", "lab", "migration-window.yaml"))
	if err != nil {
		t.Fatalf("ParseSpec() error = %v", err)
	}

	if !spec.Options.Approval.Required {
		t.Fatal("Approval.Required = false, want true")
	}
	if spec.Options.Waves.Size != 2 {
		t.Fatalf("Wave size = %d, want 2", spec.Options.Waves.Size)
	}
	if spec.Target.Platform != models.PlatformProxmox {
		t.Fatalf("Target.Platform = %q, want %q", spec.Target.Platform, models.PlatformProxmox)
	}
}

func TestParseSpec_MissingSource(t *testing.T) {
	t.Parallel()

	spec := &MigrationSpec{
		Name: "missing-source",
		Target: TargetSpec{
			Address:  "pve.lab.local",
			Platform: models.PlatformProxmox,
		},
		Workloads: []WorkloadSelector{{Match: MatchCriteria{NamePattern: "*"}}},
		Options:   MigrationOptions{Parallel: 1},
	}

	errs := ValidateSpec(spec)
	if len(errs) == 0 {
		t.Fatal("ValidateSpec() errors = empty, want validation errors")
	}
}

func TestParseSpec_InvalidPlatform(t *testing.T) {
	t.Parallel()

	spec := &MigrationSpec{
		Name: "invalid-platform",
		Source: SourceSpec{
			Address:  "source",
			Platform: models.Platform("bad"),
		},
		Target: TargetSpec{
			Address:  "target",
			Platform: models.PlatformProxmox,
		},
		Workloads: []WorkloadSelector{{Match: MatchCriteria{NamePattern: "*"}}},
		Options:   MigrationOptions{Parallel: 1},
	}

	errs := ValidateSpec(spec)
	if len(errs) == 0 {
		t.Fatal("ValidateSpec() errors = empty, want validation errors")
	}
}

func TestParseSpec_GlobMatching(t *testing.T) {
	t.Parallel()

	vms := sampleVirtualMachines()
	spec := sampleSpec()

	matched := MatchWorkloads(vms, spec.Workloads[:1])
	if len(matched) != 2 {
		t.Fatalf("len(MatchWorkloads()) = %d, want 2", len(matched))
	}
}

func TestValidateSpec_MultipleErrors(t *testing.T) {
	t.Parallel()

	spec := &MigrationSpec{
		Source: SourceSpec{
			Platform: models.Platform("unknown"),
		},
		Target: TargetSpec{
			Platform: models.Platform("also-unknown"),
		},
		Workloads: []WorkloadSelector{
			{Match: MatchCriteria{NamePattern: "["}},
		},
		Options: MigrationOptions{Parallel: 0},
	}

	errs := ValidateSpec(spec)
	if len(errs) < 4 {
		t.Fatalf("len(ValidateSpec()) = %d, want multiple errors", len(errs))
	}
}

func TestValidateSpec_InvalidExecutionWindow(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Options.Window = ExecutionWindow{
		NotBefore: time.Date(2026, time.April, 4, 18, 0, 0, 0, time.UTC),
		NotAfter:  time.Date(2026, time.April, 4, 17, 0, 0, 0, time.UTC),
	}

	errs := ValidateSpec(spec)
	if len(errs) == 0 {
		t.Fatal("ValidateSpec() errors = empty, want invalid window error")
	}
}

func TestValidateSpec_EmptyDependencyRejected(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Workloads[0].Overrides.Dependencies = []string{"db-01", ""}

	errs := ValidateSpec(spec)
	if len(errs) == 0 {
		t.Fatal("ValidateSpec() errors = empty, want dependency validation error")
	}
}
