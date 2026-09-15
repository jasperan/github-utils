package run

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jasperan/github-utils/gotui/internal/cli"
)

// invFor builds a single-command invocation for the runner tests.
//
// The env is inherited so the child sees a normal environment; these tests shell
// out to /bin/sh rather than to a checkout.
func invFor(script string) cli.Invocation {
	return cli.Invocation{Argv: []string{"sh", "-c", script}, Env: os.Environ()}
}

// TestRunCapturesOutputAndReportsTheChildsMessage runs a real child so the
// subprocess path is exercised without needing Python.
func TestRunCapturesOutputAndReportsTheChildsMessage(t *testing.T) {
	out, err := execInvocation(invFor("echo hello; echo boom >&2; exit 3"), 30*time.Second)
	if err == nil {
		t.Fatal("Run returned no error for a failing child")
	}
	if !strings.Contains(string(out), "hello") {
		t.Errorf("output = %q, want the child's stdout captured", out)
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %q, want the child's own message rather than an exit status", err)
	}
}

// TestRunSucceedsOnExitZero is the happy path.
func TestRunSucceedsOnExitZero(t *testing.T) {
	out, err := execInvocation(invFor("echo done"), 30*time.Second)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(string(out), "done") {
		t.Errorf("output = %q", out)
	}
}

// TestRunReturnsOutputOnANonZeroExit is required by the pattern catalogue: a bare
// `paint` exits 1 BY DESIGN while printing the pattern list, so the caller must
// still receive stdout.
func TestRunReturnsOutputOnANonZeroExit(t *testing.T) {
	out, err := execInvocation(invFor("echo 'Available patterns: checkerboard, heart, wave'; exit 1"), 30*time.Second)
	if err == nil {
		t.Fatal("Run returned no error for a child that exited 1")
	}
	if !strings.Contains(string(out), "checkerboard") {
		t.Errorf("output = %q, want stdout returned alongside the error", out)
	}
}

// TestRunEnforcesItsDeadline is the regression guard for the bug that made this
// package necessary.
//
// `exec.CommandContext` kills only the DIRECT child. github-utils is reached
// through wrappers that spawn children of their own, and a surviving descendant
// holds the inherited stdout pipe open, so cmd.Run() keeps waiting on the pipe
// copy and the deadline is silently skipped. Before the process-group signal and
// the pipe grace, this exact test took the child's full 30 seconds.
func TestRunEnforcesItsDeadline(t *testing.T) {
	start := time.Now()
	_, err := execInvocation(invFor("sleep 30"), 300*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run returned no error for a child that outlived its deadline")
	}
	if !strings.Contains(err.Error(), "timed out") {
		t.Errorf("error = %q, want a timeout message", err)
	}
	if elapsed > 10*time.Second {
		t.Errorf("Run took %s; the deadline was not enforced", elapsed)
	}
}

// TestRunEnforcesItsDeadlineWithAGrandchildHoldingThePipe is the sharper form of
// the same bug: the direct child exits immediately, but a backgrounded
// grandchild inherits stdout, so a wait for EOF would block until it finished.
//
// Killing the whole process group is what makes this return promptly.
func TestRunEnforcesItsDeadlineWithAGrandchildHoldingThePipe(t *testing.T) {
	start := time.Now()
	_, err := execInvocation(invFor("sleep 30 & wait"), 300*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Run returned no error for a child tree that outlived its deadline")
	}
	if elapsed > 10*time.Second {
		t.Errorf("Run took %s; a grandchild holding the pipe defeated the deadline", elapsed)
	}
}

// TestRunReportsAMissingBinary is the actionable first-run failure: the
// interpreter was resolved but does not exist.
func TestRunReportsAMissingBinary(t *testing.T) {
	inv := cli.Invocation{Argv: []string{"/nonexistent/github-utils-does-not-exist"}, Env: os.Environ()}
	_, err := execInvocation(inv, 5*time.Second)
	if err == nil {
		t.Fatal("Run returned no error for a missing binary")
	}
	if !strings.Contains(err.Error(), "not found on PATH") {
		t.Errorf("error = %q, want it to name the missing program", err)
	}
}

// TestRunRejectsAnEmptyInvocation keeps an empty argv from reaching exec.
func TestRunRejectsAnEmptyInvocation(t *testing.T) {
	if _, err := execInvocation(cli.Invocation{}, time.Second); err == nil {
		t.Fatal("Run accepted an empty argv")
	}
}

// TestSyncWriterIsSafeUnderConcurrentWrites is the race guard.
//
// os/exec reuses a single pipe only when Stdout and Stderr are the identical
// writer value. This front-end shares one writer for both, so the two copy
// goroutines write concurrently; a plain strings.Builder would corrupt, and
// `go test -race` flags it.
func TestSyncWriterIsSafeUnderConcurrentWrites(t *testing.T) {
	var w syncWriter
	done := make(chan struct{})
	for i := 0; i < 8; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 200; j++ {
				if _, err := w.Write([]byte("chunk")); err != nil {
					t.Errorf("Write: %v", err)
					return
				}
				_ = w.Bytes()
			}
		}()
	}
	for i := 0; i < 8; i++ {
		<-done
	}
	if got := len(w.Bytes()); got != 8*200*len("chunk") {
		t.Errorf("captured %d bytes, want %d", got, 8*200*len("chunk"))
	}
}

// TestFirstUsefulLinePrefersAnError pins the message a user actually needs.
func TestFirstUsefulLinePrefersAnError(t *testing.T) {
	cases := []struct {
		name, in, want string
	}{
		{"error line wins", "Error: Pattern 'nope' not found\nUsage: paint", "Error: Pattern 'nope' not found"},
		{"exception header", "Traceback\nFileNotFoundError: no such file", "FileNotFoundError: no such file"},
		{"skips pure box drawing", "┏━━━┓\n┃   ┃\nError: real problem", "Error: real problem"},
		{"keeps a table row, which has content", "┏━━━┓\n┃ 12 ┃", "┃ 12 ┃"},
		{"blank input", "   \n\n", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := firstUsefulLine(c.in); got != c.want {
				t.Errorf("firstUsefulLine(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// TestStripANSIRemovesSGR keeps a captured colour code out of an unstyled pane.
func TestStripANSIRemovesSGR(t *testing.T) {
	if got := stripANSI("\x1b[1;32mok\x1b[0m"); got != "ok" {
		t.Errorf("stripANSI = %q, want %q", got, "ok")
	}
	if got := stripANSI("plain"); got != "plain" {
		t.Errorf("stripANSI = %q, want %q", got, "plain")
	}
}
