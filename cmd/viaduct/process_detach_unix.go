//go:build !windows

package main

import (
	"os/exec"
	"syscall"
)

func configureDetachedCommand(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
