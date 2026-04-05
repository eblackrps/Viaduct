package vmware

import (
	"context"
	"testing"

	"github.com/vmware/govmomi/vim25/types"
)

func TestDistributedVLANID_NilConfig_ReturnsZero(t *testing.T) {
	t.Parallel()

	if got := distributedVLANID(types.DVPortgroupConfigInfo{}); got != 0 {
		t.Fatalf("distributedVLANID(empty) = %d, want 0", got)
	}
}

func TestExtractClusterName_EmptyReference_ReturnsEmpty(t *testing.T) {
	t.Parallel()

	if got := extractClusterName(context.Background(), nil, types.ManagedObjectReference{}); got != "" {
		t.Fatalf("extractClusterName() = %q, want empty string", got)
	}
}
