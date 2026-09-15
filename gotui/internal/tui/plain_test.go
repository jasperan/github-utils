package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jasperan/github-utils/gotui/internal/cli"
)

// TestResolvePlainRoutesEachFlag pins the flag-to-action mapping, including that
// the action order is a fixed contract rather than a guess.
func TestResolvePlainRoutesEachFlag(t *testing.T) {
	// The compare path validates that both snapshots exist, so it needs real files.
	snapshotDir := t.TempDir()
	before := writeTempFile(t, snapshotDir, "before.json", `{"total_contributions": 1}`)
	after := writeTempFile(t, snapshotDir, "after.json", `{"total_contributions": 2}`)

	cases := []struct {
		name string
		seed Seed
		want string
	}{
		{"text", Seed{Text: "HELLO"}, PlainPaintText},
		{"pattern", Seed{Pattern: "wave"}, PlainPaintPattern},
		{"simulate", Seed{Simulate: true}, PlainSimulate},
		{"compare", Seed{Before: before, After: after}, PlainCompare},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ResolvePlain(c.seed)
			if err != nil {
				t.Fatalf("ResolvePlain: %v", err)
			}
			if got.Kind != c.want {
				t.Errorf("Kind = %q, want %q", got.Kind, c.want)
			}
		})
	}
}

// writeTempFile creates a file for a test that needs a real path.
func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// TestResolvePlainPrefersTextOverEverythingElse pins the documented precedence, so
// a run that sets several flags is deterministic.
func TestResolvePlainPrefersTextOverEverythingElse(t *testing.T) {
	got, err := ResolvePlain(Seed{Text: "HI", Pattern: "wave", Simulate: true})
	if err != nil {
		t.Fatalf("ResolvePlain: %v", err)
	}
	if got.Kind != PlainPaintText {
		t.Errorf("Kind = %q, want text to win", got.Kind)
	}
}

// TestResolvePlainWithoutAnActionReportsIt keeps a bare invocation distinguishable
// from a failed one, so main can print the usage.
func TestResolvePlainWithoutAnActionReportsIt(t *testing.T) {
	_, err := ResolvePlain(Seed{})
	if !errors.Is(err, ErrNoPlainAction) {
		t.Fatalf("error = %v, want ErrNoPlainAction", err)
	}
}

// TestResolvePlainDefaultsOnlyAnOmittedDays is the bug this pins.
//
// `--simulate` on its own must use the CLI's own 90-day default, NOT be rejected for
// a flag the user never passed. A typo, by contrast, must be reported: silently
// falling back would run a different window than the user asked for.
func TestResolvePlainDefaultsOnlyAnOmittedDays(t *testing.T) {
	t.Run("omitted days uses the default", func(t *testing.T) {
		got, err := ResolvePlain(Seed{Simulate: true})
		if err != nil {
			t.Fatalf("ResolvePlain: %v", err)
		}
		if got.Days != DefaultDays {
			t.Errorf("Days = %d, want the default %d", got.Days, DefaultDays)
		}
	})

	t.Run("typed days is honoured", func(t *testing.T) {
		got, err := ResolvePlain(Seed{Simulate: true, Days: "45"})
		if err != nil {
			t.Fatalf("ResolvePlain: %v", err)
		}
		if got.Days != 45 {
			t.Errorf("Days = %d, want 45", got.Days)
		}
	})

	t.Run("a typo is an error, not a silent default", func(t *testing.T) {
		if _, err := ResolvePlain(Seed{Simulate: true, Days: "9x"}); err == nil {
			t.Fatal("ResolvePlain accepted a malformed --days")
		}
	})

	t.Run("zero days is refused", func(t *testing.T) {
		if _, err := ResolvePlain(Seed{Simulate: true, Days: "0"}); err == nil {
			t.Fatal("ResolvePlain accepted zero days")
		}
	})
}

// TestResolvePlainValidatesTypedValues keeps a bad flag from reaching the CLI.
func TestResolvePlainValidatesTypedValues(t *testing.T) {
	cases := []struct {
		name string
		seed Seed
	}{
		{"text too long", Seed{Text: "ABCDEFGHIJ"}},
		{"bad intensity", Seed{Text: "HI", Intensity: "9"}},
		{"non numeric intensity", Seed{Text: "HI", Intensity: "loud"}},
		{"bad seed", Seed{Simulate: true, Seed: "abc"}},
		{"missing before", Seed{Before: "a.json"}},
		{"missing after", Seed{After: "b.json"}},
		{"nonexistent snapshot", Seed{Before: "nope.json", After: "nope2.json"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ResolvePlain(c.seed); err == nil {
				t.Error("ResolvePlain accepted an invalid value")
			}
		})
	}
}

// TestResolvePlainSeedPointerSemantics pins that an absent seed stays nil (random
// plan) while zero is passed through.
func TestResolvePlainSeedPointerSemantics(t *testing.T) {
	random, err := ResolvePlain(Seed{Simulate: true})
	if err != nil {
		t.Fatalf("ResolvePlain: %v", err)
	}
	if random.Seed != nil {
		t.Errorf("Seed = %v, want nil for a random plan", *random.Seed)
	}

	zero, err := ResolvePlain(Seed{Simulate: true, Seed: "0"})
	if err != nil {
		t.Fatalf("ResolvePlain: %v", err)
	}
	if zero.Seed == nil || *zero.Seed != 0 {
		t.Errorf("Seed = %v, want a pointer to 0", zero.Seed)
	}
}

// TestRunPlainPrintsTheGridAndTheCLIsOutput is the pipeline path: no styling, but
// still the parsed grid and the CLI's own text.
func TestRunPlainPrintsTheGridAndTheCLIsOutput(t *testing.T) {
	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture)}
	opts := testOptions(runner)

	report, err := RunPlain(opts, PlainAction{Kind: PlainPaintText, Text: "HELLO", Intensity: 4})
	if err != nil {
		t.Fatalf("RunPlain: %v", err)
	}

	// The grid block must be present and must use the block glyphs, so the output
	// stays legible in a pipe.
	if !strings.Contains(report, "█") {
		t.Errorf("report is missing the rendered grid:\n%s", report)
	}
	if !strings.Contains(report, "total estimated commits: 544") {
		t.Errorf("report is missing the recovered totals:\n%s", report)
	}
	// The command that produced it must be shown, so a reader can reproduce it.
	if !strings.Contains(report, "paint HELLO --preview") {
		t.Errorf("report does not name the command it ran:\n%s", report)
	}
	// No colour may be emitted on this path.
	if strings.ContainsRune(report, 0x1b) {
		t.Error("the plain report contains an ANSI escape")
	}
}

// TestRunPlainPropagatesAFailure keeps a broken run from looking successful.
func TestRunPlainPropagatesAFailure(t *testing.T) {
	runner := &stubRunner{err: errors.New("Error: Pattern 'nope' not found as file or builtin pattern")}
	opts := testOptions(runner)

	report, err := RunPlain(opts, PlainAction{Kind: PlainPaintPattern, Pattern: "nope"})
	if err == nil {
		t.Fatal("RunPlain returned no error for a failed CLI call")
	}
	if !strings.Contains(report, "nope") {
		t.Errorf("report should carry the CLI's message:\n%s", report)
	}
}

// TestRunPlainSimulateAndCompareStayOnTheCLI pins that both actions use the CLI's
// own output rather than anything recomputed here.
func TestRunPlainSimulateAndCompareStayOnTheCLI(t *testing.T) {
	runner := &stubRunner{simulate: []byte(simulateFixture), other: []byte("Snapshot Comparison\nMetric Before After Delta\n")}
	opts := testOptions(runner)

	if _, err := RunPlain(opts, PlainAction{Kind: PlainSimulate, Days: 30, Profile: "default"}); err != nil {
		t.Fatalf("RunPlain simulate: %v", err)
	}
	if _, err := RunPlain(opts, PlainAction{Kind: PlainCompare, Before: "/tmp/a.json", After: "/tmp/b.json"}); err != nil {
		t.Fatalf("RunPlain compare: %v", err)
	}

	var sawSimulate, sawCompare bool
	for _, inv := range runner.recorded() {
		joined := strings.Join(inv.Argv, " ")
		if strings.Contains(joined, "simulate") {
			sawSimulate = true
			if !strings.Contains(joined, "--dry-run") {
				t.Errorf("simulate argv %v must be a dry run", inv.Argv)
			}
		}
		if strings.Contains(joined, "stats compare") {
			sawCompare = true
		}
	}
	if !sawSimulate {
		t.Error("no simulate invocation was made")
	}
	if !sawCompare {
		t.Error("no stats-compare invocation was made")
	}
}

// TestRunPlainWithoutARunnerReportsIt keeps a missing dependency visible.
func TestRunPlainWithoutARunnerReportsIt(t *testing.T) {
	opts := testOptions(nil)
	if _, err := RunPlain(opts, PlainAction{Kind: PlainSimulate, Days: 30}); err == nil {
		t.Fatal("RunPlain returned no error without a runner")
	}
}

// TestPlainUsageNamesTheFlags is the actionable refusal: a bare run on a pipe must
// tell the user what to pass, including the exact flags that exist.
func TestPlainUsageNamesTheFlags(t *testing.T) {
	opts := testOptions(&stubRunner{})
	usage := PlainUsage(opts)

	for _, want := range []string{"--paint", "--pattern", "--simulate", "--before", "--after", "--plain"} {
		if !strings.Contains(usage, want) {
			t.Errorf("usage does not mention %s:\n%s", want, usage)
		}
	}
	// Every flag the usage advertises must actually exist on the command line.
	for _, bad := range []string{"--compare"} {
		if strings.Contains(usage, bad) {
			t.Errorf("usage advertises %s, which is not a real flag", bad)
		}
	}
	if !strings.Contains(usage, opts.Launcher.Bin) {
		t.Error("usage should name the CLI it delegates to")
	}
}

// TestPlainResultOmitsAnAbsentSummary keeps an empty summary line from appearing.
func TestPlainResultOmitsAnAbsentSummary(t *testing.T) {
	got := plainResult("github-utils paint X --dry-run", []byte("Execution not yet implemented.\n"), nil)
	if strings.Contains(got, "total estimated") {
		t.Errorf("report invented a total:\n%s", got)
	}
	if !strings.Contains(got, "Execution not yet implemented.") {
		t.Errorf("report dropped the CLI's output:\n%s", got)
	}
}

// TestPlainRunnerUsesTheConfiguredTimeout keeps the deadline from silently
// defaulting when the caller set one.
func TestPlainRunnerUsesTheConfiguredTimeout(t *testing.T) {
	recording := &timeoutRecorder{}
	opts := Options{
		Launcher: cli.Launcher{Bin: "github-utils"},
		Runner:   recording,
		Timeout:  7 * time.Second,
	}
	if _, err := RunPlain(opts, PlainAction{Kind: PlainSimulate, Days: 5}); err != nil {
		t.Fatalf("RunPlain: %v", err)
	}
	if recording.got != 7*time.Second {
		t.Errorf("timeout = %s, want 7s", recording.got)
	}
}

// timeoutRecorder captures the deadline it was handed.
type timeoutRecorder struct{ got time.Duration }

func (r *timeoutRecorder) Run(_ cli.Invocation, timeout time.Duration) ([]byte, error) {
	r.got = timeout
	return []byte("Total commits: 1\n"), nil
}
