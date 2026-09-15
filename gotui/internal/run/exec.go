package run

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/jasperan/github-utils/gotui/internal/cli"
)

// pipeGrace bounds how long the wait for the output pipes may outlive the child.
//
// Killing a process does not necessarily close its stdout: a descendant that
// inherited the pipe keeps it open, and the copy goroutine then blocks until that
// descendant exits. Without this bound the deadline would be skipped entirely and
// the UI would look frozen. See procgroup_unix.go for the other half.
const pipeGrace = 5 * time.Second

// execInvocation runs one invocation under a deadline.
func execInvocation(inv cli.Invocation, timeout time.Duration) ([]byte, error) {
	if len(inv.Argv) == 0 {
		return nil, errors.New("empty invocation")
	}
	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, inv.Argv[0], inv.Argv[1:]...)
	if inv.Env != nil {
		cmd.Env = inv.Env
	}
	cmd.Dir = inv.Dir

	// exec.CommandContext kills only the direct child. This CLI is reached through
	// wrappers that spawn children of their own, so signal the whole group, and
	// bound the pipe wait in case a descendant survives long enough to hold
	// stdout open. Without both, the deadline is not enforced.
	setProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcessGroup(cmd) }
	cmd.WaitDelay = pipeGrace

	// stdout and stderr share one mutex-guarded buffer: os/exec only reuses a
	// single pipe when the two writers are the identical value, and here they are
	// separate values (stderr is teed so it can also be examined alone), so a plain
	// Builder would be written by two goroutines at once and `go test -race` would
	// flag it.
	var merged syncWriter
	var stderr strings.Builder
	cmd.Stdout = &merged
	// stderr is captured both in the merged stream (so the user sees the CLI's
	// output in order) and on its own, because the useful failure message is almost
	// always the one written to stderr rather than whatever happened to reach
	// stdout first.
	cmd.Stderr = io.MultiWriter(&merged, &stderr)

	err := cmd.Run()
	text := merged.Bytes()

	if ctx.Err() == context.DeadlineExceeded {
		return text, fmt.Errorf("timed out after %s", timeout)
	}
	if err != nil {
		if _, lookErr := exec.LookPath(inv.Argv[0]); lookErr != nil {
			return text, fmt.Errorf("%s not found on PATH: %w", inv.Argv[0], lookErr)
		}
		// A non-zero exit is reported with the child's own first useful line, which
		// is far more actionable than "exit status 1". stderr is preferred because
		// that is where a CLI reports a failure; stdout is only a fallback. The
		// output is returned either way: callers that expect a non-zero exit (the
		// pattern catalogue) read it.
		if line := firstUsefulLine(stderr.String()); line != "" {
			return text, errors.New(line)
		}
		if line := firstUsefulLine(string(text)); line != "" {
			return text, errors.New(line)
		}
		return text, err
	}
	return text, nil
}

// syncWriter is a buffer that stdout and stderr can share.
//
// Interleaving is preserved because both copy goroutines write through the one
// lock, so the captured output reads in roughly arrival order instead of being
// concatenated stream-by-stream.
type syncWriter struct {
	mu sync.Mutex
	b  strings.Builder
}

func (w *syncWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

// Bytes returns the captured output so far.
func (w *syncWriter) Bytes() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	return []byte(w.b.String())
}

// firstUsefulLine picks the most informative line of a failure's output.
//
// argparse writes usage to stderr and Rich writes a table, so blank lines and
// lines with no letters or digits are skipped. An exception header
// ("ValueError: ...") is preferred when one is present, because it is the actual
// failure rather than the command echo above it.
func firstUsefulLine(output string) string {
	fallback := ""
	for _, line := range strings.Split(strings.TrimSpace(stripANSI(output)), "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.IndexFunc(s, isWordRune) < 0 {
			continue
		}
		if strings.Contains(s, "Error:") || exceptionHeader(s) {
			return s
		}
		if fallback == "" {
			fallback = s
		}
	}
	return fallback
}

// exceptionHeader reports whether a line looks like a raised exception
// ("FileNotFoundError: ...") rather than a table row or a usage line.
func exceptionHeader(s string) bool {
	head, _, found := strings.Cut(s, ":")
	if !found || strings.ContainsAny(head, " \t│┃|") {
		return false
	}
	return strings.HasSuffix(head, "Error") || strings.HasSuffix(head, "Exception")
}

// isWordRune reports whether r is a letter or digit, used to reject lines that
// are only box-drawing characters or whitespace.
func isWordRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

// stripANSI removes SGR colour sequences so a Rich-styled line can be matched and
// displayed as plain text. Piped output normally has none, but a caller with
// FORCE_COLOR set would produce them.
func stripANSI(s string) string {
	if !strings.ContainsRune(s, 0x1b) {
		return s
	}
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case esc:
			if r == 'm' {
				esc = false
			}
		case r == 0x1b:
			esc = true
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
