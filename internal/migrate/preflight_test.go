package migrate

import (
	"context"
	"testing"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
)

func TestPreflight_AllPass(t *testing.T) {
	t.Parallel()

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		sampleSpec(),
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if !report.CanProceed {
		t.Fatalf("CanProceed = false, want true: %#v", report)
	}
}

func TestPreflight_DiskSpaceFail(t *testing.T) {
	t.Parallel()

	target := sampleTargetResult()
	target.Datastores[0].FreeMB = 1
	target.Datastores[1].FreeMB = 1

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: target.Platform, result: target},
		sampleSpec(),
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestPreflight_NameConflict(t *testing.T) {
	t.Parallel()

	target := sampleTargetResult()
	target.VMs = []models.VirtualMachine{{Name: "web-01"}}

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: target.Platform, result: target},
		sampleSpec(),
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestPreflight_NetworkMappingFail(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Workloads = spec.Workloads[:1]
	spec.Workloads[0].Overrides.NetworkMap["VM Network"] = "missing"

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		spec,
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestPreflight_BackupWarn(t *testing.T) {
	t.Parallel()

	source := sampleSourceResult()
	for index := range source.VMs {
		source.VMs[index].Snapshots = nil
	}

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: source.Platform, result: source},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		sampleSpec(),
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if !report.CanProceed {
		t.Fatal("CanProceed = false, want true")
	}
}

func TestPreflight_MultipleFailures(t *testing.T) {
	t.Parallel()

	target := sampleTargetResult()
	target.Datastores[0].FreeMB = 1
	target.Networks = []models.NetworkInfo{{Name: "vmbr0"}}
	target.VMs = []models.VirtualMachine{{Name: "web-01"}}

	spec := sampleSpec()
	spec.Workloads[0].Overrides.NetworkMap["Management"] = "missing"

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: target.Platform, result: target},
		spec,
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.FailCount < 2 {
		t.Fatalf("FailCount = %d, want at least 2", report.FailCount)
	}
}

func TestPreflight_ApprovalGateFail(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Options.Approval = ApprovalGate{Required: true}

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		spec,
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestPreflight_ExecutionWindowFail(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Options.Window = ExecutionWindow{
		NotBefore: time.Now().UTC().Add(2 * time.Hour),
	}

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		spec,
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.CanProceed {
		t.Fatal("CanProceed = true, want false")
	}
}

func TestPreflight_ExecutionPlanIncluded(t *testing.T) {
	t.Parallel()

	spec := sampleSpec()
	spec.Workloads[0].Overrides.Dependencies = []string{"db-01"}
	spec.Workloads = append(spec.Workloads, WorkloadSelector{
		Match: MatchCriteria{NamePattern: "db-*"},
	})

	checker := NewPreflightChecker(
		&mockMigrationConnector{platform: sampleSourceResult().Platform, result: sampleSourceResult()},
		&mockMigrationConnector{platform: sampleTargetResult().Platform, result: sampleTargetResult()},
		spec,
	)

	report, err := checker.RunAll(context.Background())
	if err != nil {
		t.Fatalf("RunAll() error = %v", err)
	}
	if report.Plan == nil || len(report.Plan.Waves) == 0 {
		t.Fatalf("Plan = %#v, want planned waves", report.Plan)
	}
}
