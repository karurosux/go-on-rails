//go:build windows

package run

import "os/exec"

// Process groups are a Unix concept; on Windows the child is signaled directly.

func setProcAttrs(*exec.Cmd) {}

func interruptProc(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill() // os.Interrupt is not implemented on Windows
	}
}

func killProc(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
