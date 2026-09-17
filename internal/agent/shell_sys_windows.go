//go:build windows

package agent

import (
	"os"
	"os/exec"
	"syscall"
)

func shellSetpgidAttr() *syscall.SysProcAttr {
	return nil
}

func shellKillGroup(cmd *exec.Cmd, signal syscall.Signal) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return cmd.Process.Kill() == nil
}

func shellTerminateGroup(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return cmd.Process.Kill() == nil
}

func shellForceKillGroup(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return cmd.Process.Kill() == nil
}

func isSetuidRoot(info os.FileInfo) bool {
	return false
}
