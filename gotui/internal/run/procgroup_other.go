//go:build !unix

package run

import "os/exec"

// setProcessGroup is a no-op where process groups are unavailable.
//
// WaitDelay still bounds how long Run waits on the pipes, so the deadline stays
// honoured even without a group to signal.
func setProcessGroup(*exec.Cmd) {}

// killProcessGroup falls back to killing the direct child.
func killProcessGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}
