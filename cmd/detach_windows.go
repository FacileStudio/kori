//go:build windows

package cmd

import "os/exec"

func setDetachedProcAttr(cmd *exec.Cmd) {}
