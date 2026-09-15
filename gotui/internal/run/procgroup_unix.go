//go:build unix

package run

import (
	"os/exec"
	"syscall"
)

// setProcessGroup puts the child in its own process group so a cancellation can
// take down whatever it spawned.
//
// github-utils is reached through wrappers (a checkout's .venv/bin/github-utils,
// or `uv run github-utils`), and those spawn children of their own. Killing only
// the direct child would leave a descendant holding the inherited stdout pipe
// open, so the copy goroutine would never see EOF and Run would block until that
// descendant exited on its own -- the deadline silently skipped, and the UI left
// looking frozen.
func setProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup signals the whole group, which closes inherited pipes
// immediately instead of waiting for descendants to exit.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		return cmd.Process.Kill()
	}
	return nil
}
