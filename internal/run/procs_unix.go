//go:build !windows

package run

import (
	"os/exec"
	"syscall"
)

// setProcAttrs puts the child in its own process group, so interrupting or
// killing it reaches its whole tree (air → the compiled server, npm → vite).
// Without this, killing the wrapper can orphan the real server, which then
// squats on the port for the next `gor dev`.
func setProcAttrs(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// interruptProc sends SIGINT to the child's process group, falling back to
// the child alone when the group signal fails.
func interruptProc(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGINT); err != nil {
		_ = cmd.Process.Signal(syscall.SIGINT)
	}
}

// killProc force-kills the child's process group, falling back to the child
// alone when the group signal fails.
func killProc(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		_ = cmd.Process.Kill()
	}
}
