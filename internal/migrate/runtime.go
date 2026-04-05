package migrate

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/eblackrps/viaduct/internal/models"
	"github.com/eblackrps/viaduct/internal/store"
)

type diskExporter interface {
	ExportVMDisks(ctx context.Context, vm models.VirtualMachine) ([]string, error)
}

type vmPowerController interface {
	PowerOffVM(ctx context.Context, vmID string) error
	PowerOnVM(ctx context.Context, vmID string) error
	RestoreVMPowerState(ctx context.Context, vmID string, state models.PowerState) error
}

type vmImporter interface {
	CreateVM(ctx context.Context, vm models.VirtualMachine, convertedDisks []string, targetHost, targetStorage string) (string, error)
}

type vmNetworkConfigurer interface {
	ConfigureVMNetworks(ctx context.Context, vmID string, nics []MappedNIC) error
}

type vmVerifier interface {
	VerifyVM(ctx context.Context, vmID string) error
}

type vmRemover interface {
	RemoveVM(ctx context.Context, vmID string) error
}

type vmSnapshotter interface {
	CreateVMSnapshot(ctx context.Context, vmID string) (string, error)
}

type diskConvertFunc func(ctx context.Context, req ConversionRequest) (*ConversionResult, error)

func persistMigrationState(ctx context.Context, stateStore store.Store, state *MigrationState) error {
	if stateStore == nil {
		return nil
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("persist migration state: %w", err)
	}

	record := store.MigrationRecord{
		ID:          state.ID,
		TenantID:    store.TenantIDFromContext(ctx),
		SpecName:    state.SpecName,
		Phase:       string(state.Phase),
		StartedAt:   state.StartedAt,
		UpdatedAt:   state.UpdatedAt,
		CompletedAt: state.CompletedAt,
		RawJSON:     payload,
	}
	if err := stateStore.SaveMigration(ctx, store.TenantIDFromContext(ctx), record); err != nil {
		return fmt.Errorf("persist migration state: %w", err)
	}

	return nil
}

func loadMigrationState(ctx context.Context, stateStore store.Store, migrationID string) (*MigrationState, error) {
	if stateStore == nil {
		return nil, fmt.Errorf("load migration state: store is nil")
	}

	record, err := stateStore.GetMigration(ctx, store.TenantIDFromContext(ctx), migrationID)
	if err != nil {
		return nil, fmt.Errorf("load migration state: %w", err)
	}

	var state MigrationState
	if err := json.Unmarshal(record.RawJSON, &state); err != nil {
		return nil, fmt.Errorf("load migration state: decode %s: %w", migrationID, err)
	}

	return &state, nil
}

func persistRecoveryPoint(ctx context.Context, stateStore store.Store, point *RecoveryPoint) error {
	if stateStore == nil {
		return nil
	}

	payload, err := json.Marshal(point)
	if err != nil {
		return fmt.Errorf("persist recovery point: %w", err)
	}

	record := store.RecoveryPointRecord{
		MigrationID: point.MigrationID,
		TenantID:    store.TenantIDFromContext(ctx),
		Phase:       string(point.Phase),
		CreatedAt:   point.CreatedAt,
		RawJSON:     payload,
	}
	if err := stateStore.SaveRecoveryPoint(ctx, store.TenantIDFromContext(ctx), record); err != nil {
		return fmt.Errorf("persist recovery point: %w", err)
	}

	return nil
}

func loadRecoveryPoint(ctx context.Context, stateStore store.Store, migrationID string) (*RecoveryPoint, error) {
	if stateStore == nil {
		return nil, fmt.Errorf("load recovery point: store is nil")
	}

	record, err := stateStore.GetRecoveryPoint(ctx, store.TenantIDFromContext(ctx), migrationID)
	if err != nil {
		return nil, fmt.Errorf("load recovery point: %w", err)
	}

	var point RecoveryPoint
	if err := json.Unmarshal(record.RawJSON, &point); err != nil {
		return nil, fmt.Errorf("load recovery point: decode %s: %w", migrationID, err)
	}

	return &point, nil
}

func targetDiskFormat(platform models.Platform) DiskFormat {
	switch platform {
	case models.PlatformVMware:
		return FormatVMDK
	case models.PlatformHyperV:
		return FormatVHDX
	case models.PlatformProxmox, models.PlatformKVM:
		return FormatQCOW2
	default:
		return FormatRAW
	}
}

func inferDiskFormat(path string, platform models.Platform) DiskFormat {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".vmdk":
		return FormatVMDK
	case ".qcow2":
		return FormatQCOW2
	case ".vhd":
		return FormatVHD
	case ".vhdx":
		return FormatVHDX
	case ".raw", ".img":
		return FormatRAW
	}

	switch platform {
	case models.PlatformVMware:
		return FormatVMDK
	case models.PlatformHyperV:
		return FormatVHDX
	case models.PlatformProxmox, models.PlatformKVM:
		return FormatQCOW2
	default:
		return FormatRAW
	}
}

func deriveTargetDiskPath(path string, format DiskFormat) string {
	base := strings.TrimSuffix(path, filepath.Ext(path))
	switch format {
	case FormatVMDK:
		return base + ".vmdk"
	case FormatQCOW2:
		return base + ".qcow2"
	case FormatVHD:
		return base + ".vhd"
	case FormatVHDX:
		return base + ".vhdx"
	default:
		return base + ".raw"
	}
}

func nowUTC() time.Time {
	return time.Now().UTC()
}
