//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

func runtimeProcessMatchesCurrentExecutable(pid int) (bool, string) {
	current, err := os.Executable()
	if err != nil {
		return false, fmt.Sprintf("Unable to verify the current executable: %v.", err)
	}
	current = cleanExecutablePath(current)

	if runtime.GOOS == "linux" {
		target, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid))
		if err != nil {
			return false, fmt.Sprintf("Unable to verify process %d executable: %v.", pid, err)
		}
		if !sameViaductExecutableIdentity(cleanExecutablePath(target), current) {
			return false, fmt.Sprintf("Process %d is not the recorded Viaduct executable.", pid)
		}
		return true, "Process executable matches."
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return false, fmt.Sprintf("Unable to find process %d: %v.", pid, err)
	}
	if err := process.Signal(syscall.Signal(0)); err != nil {
		return false, fmt.Sprintf("Process %d is not running: %v.", pid, err)
	}
	return true, "Process is running; executable path verification is unavailable on this platform."
}

func cleanExecutablePath(path string) string {
	evaluated, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = evaluated
	}
	absolute, err := filepath.Abs(path)
	if err == nil {
		path = absolute
	}
	return filepath.Clean(path)
}

func sameViaductExecutableIdentity(target, current string) bool {
	if target == current {
		return true
	}
	targetBase := strings.ToLower(filepath.Base(target))
	currentBase := strings.ToLower(filepath.Base(current))
	return targetBase == currentBase && strings.Contains(currentBase, "viaduct")
}
