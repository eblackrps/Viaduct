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

func TestValidateConversion_MissingFile(t *testing.T) {
	t.Parallel()

	err := ValidateConversion("source.vmdk", filepath.Join(t.TempDir(), "missing.qcow2"))
	if err == nil {
		t.Fatal("ValidateConversion() error = nil, want error")
	}
}
