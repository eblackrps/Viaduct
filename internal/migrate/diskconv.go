package migrate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// DiskFormat identifies a supported virtual disk image format.
type DiskFormat string

const (
	// FormatVMDK represents VMware VMDK disk images.
	FormatVMDK DiskFormat = "vmdk"
	// FormatQCOW2 represents QCOW2 disk images.
	FormatQCOW2 DiskFormat = "qcow2"
	// FormatVHD represents legacy Microsoft VHD disk images.
	FormatVHD DiskFormat = "vhd"
	// FormatVHDX represents Microsoft VHDX disk images.
	FormatVHDX DiskFormat = "vhdx"
	// FormatRAW represents raw disk images.
	FormatRAW DiskFormat = "raw"
)

// ConversionRequest describes a disk conversion operation.
type ConversionRequest struct {
	SourcePath   string
	SourceFormat DiskFormat
	TargetPath   string
	TargetFormat DiskFormat
	Thin         bool
	OnProgress   func(percent int)
}

// ConversionResult captures conversion output metadata and integrity details.
type ConversionResult struct {
	SourcePath      string
	TargetPath      string
	SourceFormat    DiskFormat
	TargetFormat    DiskFormat
	SourceSizeBytes int64
	TargetSizeBytes int64
	Duration        time.Duration
	SourceChecksum  string
	TargetChecksum  string
}

// ConvertDisk converts a virtual disk between supported formats by shelling out to qemu-img.
func ConvertDisk(ctx context.Context, req ConversionRequest) (*ConversionResult, error) {
	if strings.TrimSpace(req.SourcePath) == "" {
		return nil, fmt.Errorf("convert disk: source path is required")
	}
	if strings.TrimSpace(req.TargetPath) == "" {
		return nil, fmt.Errorf("convert disk: target path is required")
	}

	sourceInfo, err := os.Stat(req.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("convert disk: stat source %s: %w", req.SourcePath, err)
	}

	sourceFormat, err := qemuFormat(req.SourceFormat)
	if err != nil {
		return nil, fmt.Errorf("convert disk: source format: %w", err)
	}

	targetFormat, err := qemuFormat(req.TargetFormat)
	if err != nil {
		return nil, fmt.Errorf("convert disk: target format: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(req.TargetPath), 0o755); err != nil {
		return nil, fmt.Errorf("convert disk: create target directory: %w", err)
	}

	if req.OnProgress != nil {
		req.OnProgress(0)
	}

	startedAt := time.Now()
	args := []string{"convert", "-f", sourceFormat, "-O", targetFormat}
	if req.Thin {
		args = append(args, "-o", "preallocation=off")
	}
	args = append(args, req.SourcePath, req.TargetPath)

	command := exec.CommandContext(ctx, "qemu-img", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("convert disk: qemu-img convert: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	sourceChecksum, err := checksumFile(req.SourcePath, req.OnProgress, 0, 50)
	if err != nil {
		return nil, fmt.Errorf("convert disk: checksum source: %w", err)
	}

	targetChecksum, err := checksumFile(req.TargetPath, req.OnProgress, 50, 100)
	if err != nil {
		return nil, fmt.Errorf("convert disk: checksum target: %w", err)
	}

	targetInfo, err := os.Stat(req.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("convert disk: stat target %s: %w", req.TargetPath, err)
	}

	if req.OnProgress != nil {
		req.OnProgress(100)
	}

	return &ConversionResult{
		SourcePath:      req.SourcePath,
		TargetPath:      req.TargetPath,
		SourceFormat:    req.SourceFormat,
		TargetFormat:    req.TargetFormat,
		SourceSizeBytes: sourceInfo.Size(),
		TargetSizeBytes: targetInfo.Size(),
		Duration:        time.Since(startedAt),
		SourceChecksum:  sourceChecksum,
		TargetChecksum:  targetChecksum,
	}, nil
}

// ValidateConversion runs qemu-img check against the converted target disk.
func ValidateConversion(source, target string) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("validate conversion: target path is required")
	}
	if _, err := os.Stat(target); err != nil {
		return fmt.Errorf("validate conversion: stat target %s: %w", target, err)
	}

	args := []string{"check", target}
	command := exec.Command("qemu-img", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("validate conversion: qemu-img check %s: %w: %s", target, err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

func qemuFormat(format DiskFormat) (string, error) {
	switch format {
	case FormatVMDK:
		return "vmdk", nil
	case FormatQCOW2:
		return "qcow2", nil
	case FormatVHD:
		return "vpc", nil
	case FormatVHDX:
		return "vhdx", nil
	case FormatRAW:
		return "raw", nil
	default:
		return "", fmt.Errorf("unsupported disk format %q", format)
	}
}

func checksumFile(path string, callback func(percent int), startPercent, endPercent int) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open %s: %w", path, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", path, err)
	}

	hasher := sha256.New()
	size := info.Size()
	if size <= 0 {
		if _, err := io.Copy(hasher, file); err != nil {
			return "", fmt.Errorf("hash %s: %w", path, err)
		}
		return hex.EncodeToString(hasher.Sum(nil)), nil
	}

	buffer := make([]byte, 1024*1024)
	var readBytes int64
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			if _, err := hasher.Write(buffer[:n]); err != nil {
				return "", fmt.Errorf("hash %s: %w", path, err)
			}
			readBytes += int64(n)
			if callback != nil {
				span := endPercent - startPercent
				percent := startPercent + int((float64(readBytes)/float64(size))*float64(span))
				if percent > endPercent {
					percent = endPercent
				}
				callback(percent)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", fmt.Errorf("hash %s: %w", path, readErr)
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}
