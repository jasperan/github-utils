// Package run executes a prepared github-utils invocation.
//
// It exists so every call is bounded. The CLI is reached through wrappers
// (a checkout's .venv/bin/github-utils, or `uv run github-utils`), and those
// spawn children of their own, so an unbounded call could leave a descendant
// holding the pipe open long after the front-end gave up. See exec.go for how
// that is handled.
package run

import (
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/jasperan/github-utils/gotui/internal/cli"
)

// DefaultTimeout bounds a single delegated call.
//
// Every invocation this front-end makes is a read-only listing, preview or
// plan: none touches the network or runs a simulation. They are quick, so a
// short bound is enough to turn a wedged child into a visible error.
const DefaultTimeout = 60 * time.Second

// Runner executes a prepared invocation and returns its combined output.
//
// It is an interface so the UI can be tested without a checkout or a subprocess.
type Runner interface {
	Run(inv cli.Invocation, timeout time.Duration) ([]byte, error)
}

// ShellExec runs the real CLI.
type ShellExec struct{}

// ErrNotRun is returned when no runner was supplied.
var ErrNotRun = errors.New("no runner configured")

// Run executes the invocation and returns its combined output.
//
// Output is returned even when the call fails, because one invocation depends on
// it: a bare `paint` exits 1 by design while printing the pattern catalogue on
// stdout. Callers that only want a failure message can ignore the bytes.
func (ShellExec) Run(inv cli.Invocation, timeout time.Duration) ([]byte, error) {
	return execInvocation(inv, timeout)
}

// CommandLine renders an invocation for display in the UI.
func CommandLine(inv cli.Invocation) string { return strings.Join(inv.Argv, " ") }

// LookPath is exposed so a caller can explain a missing binary.
func LookPath(name string) error {
	_, err := exec.LookPath(name)
	return err
}
