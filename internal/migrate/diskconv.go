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
	if !sourceInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("convert disk: source %s is not a regular file", req.SourcePath)
	}
	if err := validateConversionPaths(req.SourcePath, req.TargetPath); err != nil {
		return nil, fmt.Errorf("convert disk: %w", err)
	}
	if _, err := os.Stat(req.TargetPath); err == nil {
		return nil, fmt.Errorf("convert disk: target %s already exists", req.TargetPath)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("convert disk: stat target %s: %w", req.TargetPath, err)
	}

	sourceFormat, err := qemuFormat(req.SourceFormat)
	if err != nil {
		return nil, fmt.Errorf("convert disk: source format: %w", err)
	}

	targetFormat, err := qemuFormat(req.TargetFormat)
	if err != nil {
		return nil, fmt.Errorf("convert disk: target format: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(req.TargetPath), 0o750); err != nil {
		return nil, fmt.Errorf("convert disk: create target directory: %w", err)
	}
	tempTarget, err := reserveConversionTarget(req.TargetPath)
	if err != nil {
		return nil, fmt.Errorf("convert disk: reserve target: %w", err)
	}
	defer func() {
		if tempTarget != req.TargetPath {
			_ = os.Remove(tempTarget)
		}
	}()

	if req.OnProgress != nil {
		req.OnProgress(0)
	}

	startedAt := time.Now()
	args := []string{"convert", "-f", sourceFormat, "-O", targetFormat}
	if req.Thin {
		args = append(args, "-o", "preallocation=off")
	}
	args = append(args, req.SourcePath, tempTarget)

	// #nosec G204 -- qemu-img is a fixed executable and arguments are structured from validated conversion inputs.
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

	targetChecksum, err := checksumFile(tempTarget, req.OnProgress, 50, 100)
	if err != nil {
		return nil, fmt.Errorf("convert disk: checksum target: %w", err)
	}

	targetInfo, err := os.Stat(tempTarget)
	if err != nil {
		return nil, fmt.Errorf("convert disk: stat target %s: %w", tempTarget, err)
	}
	if err := os.Link(tempTarget, req.TargetPath); err != nil {
		return nil, fmt.Errorf("convert disk: finalize target %s: %w", req.TargetPath, err)
	}
	if err := os.Remove(tempTarget); err != nil {
		return nil, fmt.Errorf("convert disk: remove temporary target %s: %w", tempTarget, err)
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
	return ValidateConversionContext(context.Background(), source, target)
}

// ValidateConversionContext runs qemu-img check against the converted target disk with caller cancellation.
func ValidateConversionContext(ctx context.Context, source, target string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("validate conversion: target path is required")
	}
	if strings.TrimSpace(source) != "" {
		if err := validateConversionPaths(source, target); err != nil {
			return fmt.Errorf("validate conversion: %w", err)
		}
	}
	targetInfo, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("validate conversion: stat target %s: %w", target, err)
	}
	if !targetInfo.Mode().IsRegular() {
		return fmt.Errorf("validate conversion: target %s is not a regular file", target)
	}

	args := []string{"check", target}
	// #nosec G204 -- qemu-img is a fixed executable and target is passed as an argument rather than through a shell.
	command := exec.CommandContext(ctx, "qemu-img", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("validate conversion: qemu-img check %s: %w: %s", target, err, strings.TrimSpace(stderr.String()))
	}

	return nil
}

func validateConversionPaths(source, target string) error {
	sourcePath, err := filepath.Abs(source)
	if err != nil {
		return fmt.Errorf("resolve source path %s: %w", source, err)
	}
	targetPath, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve target path %s: %w", target, err)
	}
	if filepath.Clean(sourcePath) == filepath.Clean(targetPath) {
		return fmt.Errorf("source and target paths must be different")
	}
	return nil
}

func reserveConversionTarget(target string) (string, error) {
	dir := filepath.Dir(target)
	base := "." + filepath.Base(target) + "-*.tmp"
	file, err := os.CreateTemp(dir, base)
	if err != nil {
		return "", err
	}
	name := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return "", err
	}
	if err := os.Remove(name); err != nil {
		return "", err
	}
	return name, nil
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
	// #nosec G304 -- checksuming reads the explicit source or target disk path selected for the conversion workflow.
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
