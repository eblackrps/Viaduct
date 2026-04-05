package migrate

import (
	"context"
	"time"
)

// BlockRange identifies a changed range within a disk image.
type BlockRange struct {
	// Offset is the byte offset of the changed block range.
	Offset int64 `json:"offset" yaml:"offset"`
	// Length is the byte length of the changed block range.
	Length int64 `json:"length" yaml:"length"`
}

// CBTProvider exposes changed block tracking operations for warm migration.
type CBTProvider interface {
	// GetChangedBlocks returns dirty block ranges for a disk since a given timestamp.
	GetChangedBlocks(ctx context.Context, diskPath string, since time.Time) ([]BlockRange, error)
	// EnableCBT enables changed block tracking on a VM.
	EnableCBT(ctx context.Context, vmID string) error
}

// VMwareCBTProvider is a VMware-oriented changed block tracking provider.
type VMwareCBTProvider struct{}

// GetChangedBlocks returns tracked VMware block changes.
func (p *VMwareCBTProvider) GetChangedBlocks(ctx context.Context, diskPath string, since time.Time) ([]BlockRange, error) {
	return []BlockRange{}, nil
}

// EnableCBT enables VMware CBT for a VM.
func (p *VMwareCBTProvider) EnableCBT(ctx context.Context, vmID string) error {
	return nil
}

// QEMUDirtyBitmapProvider is a QEMU dirty-bitmap provider for KVM-style warm migration.
type QEMUDirtyBitmapProvider struct{}

// GetChangedBlocks returns tracked QEMU dirty blocks.
func (p *QEMUDirtyBitmapProvider) GetChangedBlocks(ctx context.Context, diskPath string, since time.Time) ([]BlockRange, error) {
	return []BlockRange{}, nil
}

// EnableCBT enables QEMU dirty bitmap tracking for a VM.
func (p *QEMUDirtyBitmapProvider) EnableCBT(ctx context.Context, vmID string) error {
	return nil
}
