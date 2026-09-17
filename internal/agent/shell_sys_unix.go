//go:build !windows

package agent

import (
	"os"
	"os/exec"
	"syscall"
)

func shellSetpgidAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

func shellKillGroup(cmd *exec.Cmd, signal syscall.Signal) bool {
	if cmd == nil || cmd.Process == nil {
		return false
	}
	return syscall.Kill(-cmd.Process.Pid, signal) == nil
}

func shellTerminateGroup(cmd *exec.Cmd) bool {
	return shellKillGroup(cmd, syscall.SIGTERM)
}

func shellForceKillGroup(cmd *exec.Cmd) bool {
	return shellKillGroup(cmd, syscall.SIGKILL)
}

func isSetuidRoot(info os.FileInfo) bool {
	sys, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode()&os.ModeSetuid != 0 && sys.Uid == 0
}
