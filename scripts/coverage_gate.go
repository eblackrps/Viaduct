package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fail("usage: go run ./scripts/coverage_gate.go <coverage-file> <minimum-percent>")
	}

	coverageFile := os.Args[1]
	minimum, err := strconv.ParseFloat(os.Args[2], 64)
	if err != nil {
		fail(fmt.Sprintf("parse minimum coverage %q: %v", os.Args[2], err))
	}

	// #nosec G204 -- this helper invokes the trusted Go toolchain directly and passes the coverage file as an argument.
	command := exec.Command("go", "tool", "cover", "-func="+coverageFile)
	output, err := command.CombinedOutput()
	if err != nil {
		fail(fmt.Sprintf("go tool cover failed: %v\n%s", err, bytes.TrimSpace(output)))
	}

	total, err := parseTotalCoverage(string(output))
	if err != nil {
		fail(err.Error())
	}
	if total < minimum {
		fail(fmt.Sprintf("coverage %.1f%% is below required %.1f%%", total, minimum))
	}

	fmt.Printf("coverage gate passed: %.1f%% >= %.1f%%\n", total, minimum)
}

func parseTotalCoverage(output string) (float64, error) {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "total:") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			break
		}
		percent := strings.TrimSuffix(fields[len(fields)-1], "%")
		value, err := strconv.ParseFloat(percent, 64)
		if err != nil {
			return 0, fmt.Errorf("parse total coverage from %q: %w", line, err)
		}
		return value, nil
	}
	return 0, fmt.Errorf("total coverage line not found")
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
