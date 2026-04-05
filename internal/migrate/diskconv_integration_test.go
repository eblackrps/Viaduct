//go:build integration

package migrate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestConvertDisk_Integration_ConvertsRawToRaw(t *testing.T) {
	if _, err := exec.LookPath("qemu-img"); err != nil {
		t.Skip("qemu-img is not installed")
	}

	source := filepath.Join(t.TempDir(), "source.raw")
	target := filepath.Join(t.TempDir(), "target.raw")
	if err := os.WriteFile(source, make([]byte, 1024), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   source,
		SourceFormat: FormatRAW,
		TargetPath:   target,
		TargetFormat: FormatRAW,
	}); err != nil {
		t.Fatalf("ConvertDisk() error = %v", err)
	}
}
