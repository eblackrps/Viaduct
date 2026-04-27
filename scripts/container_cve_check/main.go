package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	var (
		image      string
		severity   string
		trivyImage string
		timeout    time.Duration
	)
	flag.StringVar(&image, "image", "", "Container image reference to scan")
	flag.StringVar(&severity, "severity", "HIGH,CRITICAL", "Comma-separated severity list")
	flag.StringVar(&trivyImage, "trivy-image", "aquasec/trivy:0.63.0", "Trivy container image to run")
	flag.DurationVar(&timeout, "timeout", 10*time.Minute, "Maximum scan duration")
	flag.Parse()

	if err := run(image, severity, trivyImage, timeout); err != nil {
		fmt.Fprintf(os.Stderr, "container-cve-check: %v\n", err)
		os.Exit(1)
	}
}

func run(image, severity, trivyImage string, timeout time.Duration) error {
	image = strings.TrimSpace(image)
	severity = strings.TrimSpace(severity)
	trivyImage = strings.TrimSpace(trivyImage)
	if image == "" {
		return fmt.Errorf("image is required")
	}
	if severity == "" {
		return fmt.Errorf("severity is required")
	}
	if trivyImage == "" {
		return fmt.Errorf("trivy image is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	args := []string{
		"run",
		"--rm",
		trivyImage,
		"image",
		"--severity", severity,
		"--exit-code", "1",
		image,
	}
	// #nosec G204 -- docker is a fixed executable and scan options are structured arguments.
	command := exec.CommandContext(ctx, "docker", args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("scan %s timed out: %w", image, ctx.Err())
		}
		return fmt.Errorf("scan %s: %w", image, err)
	}
	fmt.Printf("container CVE check passed for %s (%s)\n", image, severity)
	return nil
}
