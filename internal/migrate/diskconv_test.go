package migrate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskFormatConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		format DiskFormat
		want   string
	}{
		{name: "vmdk", format: FormatVMDK, want: "vmdk"},
		{name: "qcow2", format: FormatQCOW2, want: "qcow2"},
		{name: "vhd", format: FormatVHD, want: "vhd"},
		{name: "vhdx", format: FormatVHDX, want: "vhdx"},
		{name: "raw", format: FormatRAW, want: "raw"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := string(tt.format); got != tt.want {
				t.Fatalf("string(%s) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestConvertDisk_InvalidSourcePath(t *testing.T) {
	t.Parallel()

	_, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   filepath.Join(t.TempDir(), "missing.vmdk"),
		SourceFormat: FormatVMDK,
		TargetPath:   filepath.Join(t.TempDir(), "target.qcow2"),
		TargetFormat: FormatQCOW2,
	})
	if err == nil {
		t.Fatal("ConvertDisk() error = nil, want error")
	}
}

func TestConvertDisk_UnsupportedFormat(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.img")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   source,
		SourceFormat: DiskFormat("made-up"),
		TargetPath:   filepath.Join(t.TempDir(), "target.qcow2"),
		TargetFormat: FormatQCOW2,
	})
	if err == nil {
		t.Fatal("ConvertDisk() error = nil, want error")
	}
}

func TestConvertDisk_SourceDirectoryRejected(t *testing.T) {
	t.Parallel()

	_, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   t.TempDir(),
		SourceFormat: FormatRAW,
		TargetPath:   filepath.Join(t.TempDir(), "target.qcow2"),
		TargetFormat: FormatQCOW2,
	})
	if err == nil {
		t.Fatal("ConvertDisk() error = nil, want error")
	}
}

func TestConvertDisk_SourceTargetCollisionRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "source.raw")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	_, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   source,
		SourceFormat: FormatRAW,
		TargetPath:   source,
		TargetFormat: FormatQCOW2,
	})
	if err == nil {
		t.Fatal("ConvertDisk() error = nil, want source/target collision error")
	}
}

func TestConvertDisk_ExistingTargetRejected(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	source := filepath.Join(root, "source.raw")
	target := filepath.Join(root, "target.qcow2")
	if err := os.WriteFile(source, []byte("source"), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}
	if err := os.WriteFile(target, []byte("target"), 0o644); err != nil {
		t.Fatalf("WriteFile(target) error = %v", err)
	}

	_, err := ConvertDisk(context.Background(), ConversionRequest{
		SourcePath:   source,
		SourceFormat: FormatRAW,
		TargetPath:   target,
		TargetFormat: FormatQCOW2,
	})
	if err == nil {
		t.Fatal("ConvertDisk() error = nil, want existing target error")
	}
}

func TestValidateConversion_MissingFile(t *testing.T) {
	t.Parallel()

	err := ValidateConversionContext(context.Background(), "source.vmdk", filepath.Join(t.TempDir(), "missing.qcow2"))
	if err == nil {
		t.Fatal("ValidateConversion() error = nil, want error")
	}
}
