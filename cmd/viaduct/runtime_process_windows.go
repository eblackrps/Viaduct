//go:build windows

package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func runtimeProcessMatchesCurrentExecutable(pid int) (bool, string) {
	current, err := os.Executable()
	if err != nil {
		return false, fmt.Sprintf("Unable to verify the current executable: %v.", err)
	}
	current = cleanExecutablePath(current)

	// #nosec G204 -- PowerShell is a fixed executable and the process ID is parsed as an integer before interpolation.
	command := exec.Command(
		"powershell",
		"-NoProfile",
		"-Command",
		"(Get-CimInstance Win32_Process -Filter \"ProcessId = "+strconv.Itoa(pid)+"\").ExecutablePath",
	)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return false, fmt.Sprintf("Unable to verify process %d executable: %v %s.", pid, err, strings.TrimSpace(stderr.String()))
	}

	target := cleanExecutablePath(strings.TrimSpace(stdout.String()))
	if target == "" {
		return false, fmt.Sprintf("Process %d is not running or has no executable path.", pid)
	}
	if !sameViaductExecutableIdentity(target, current) {
		return false, fmt.Sprintf("Process %d is not the recorded Viaduct executable.", pid)
	}
	return true, "Process executable matches."
}

func cleanExecutablePath(path string) string {
	absolute, err := filepath.Abs(strings.TrimSpace(path))
	if err == nil {
		path = absolute
	}
	return filepath.Clean(path)
}

func sameViaductExecutableIdentity(target, current string) bool {
	if strings.EqualFold(target, current) {
		return true
	}
	targetBase := strings.ToLower(filepath.Base(target))
	currentBase := strings.ToLower(filepath.Base(current))
	return targetBase == currentBase && strings.Contains(currentBase, "viaduct")
}
