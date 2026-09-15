package parse

import (
	"strings"
	"testing"
)

// These fixtures are captured verbatim from the CLI, so the regexes are pinned
// against real output rather than an idealised shape.

// paintDryRunTail is the end of `github-utils paint HELLO --dry-run`.
const paintDryRunTail = `Total estimated commits: 544
Active days: 136
`

// simulateDryRunTail is the end of `github-utils simulate --days 30 --dry-run`.
const simulateDryRunTail = `Total commits: 92
Active days: 26 / 30
Average per active day: 3.5
`

// patternCatalog is `github-utils paint` with no input source, which exits 1 while
// advertising the built-in patterns.
const patternCatalog = `Error: Provide text argument, --pattern, or --file.

Available patterns: checkerboard, heart, wave
`

// simulateHelpLine is the --profile row of `github-utils simulate --help`, whose
// help string is built from available_profiles().
const simulateHelpLine = `│ --profile  -p      TEXT                  Activity profile (default, night-owl, weekend-warrior, 9-to-5, open-source-maintainer). [default: default]`

// TestSummaryOfPaint pins the paint totals, including the absent average.
func TestSummaryOfPaint(t *testing.T) {
	s := SummaryOf([]byte(paintDryRunTail))

	if s.TotalCommits != 544 {
		t.Errorf("TotalCommits = %d, want 544", s.TotalCommits)
	}
	if s.ActiveDays != 136 {
		t.Errorf("ActiveDays = %d, want 136", s.ActiveDays)
	}
	if s.WindowDays != 0 {
		t.Errorf("WindowDays = %d, want 0 for a bare count", s.WindowDays)
	}
	if s.HasAvg {
		t.Error("HasAvg = true, but paint prints no average")
	}
	if s.SimulatedCommits != 0 {
		t.Errorf("SimulatedCommits = %d, want 0 for a paint plan", s.SimulatedCommits)
	}
}

// TestSummaryOfSimulate pins the simulate totals, including the window form of
// the active-days line, which carries two numbers.
func TestSummaryOfSimulate(t *testing.T) {
	s := SummaryOf([]byte(simulateDryRunTail))

	if s.SimulatedCommits != 92 {
		t.Errorf("SimulatedCommits = %d, want 92", s.SimulatedCommits)
	}
	if s.ActiveDays != 26 {
		t.Errorf("ActiveDays = %d, want 26", s.ActiveDays)
	}
	if s.WindowDays != 30 {
		t.Errorf("WindowDays = %d, want 30", s.WindowDays)
	}
	if !s.HasAvg || s.AvgPerActiveDay != 3.5 {
		t.Errorf("avg = %v (present=%v), want 3.5", s.AvgPerActiveDay, s.HasAvg)
	}
	if s.TotalCommits != 0 {
		t.Errorf("TotalCommits = %d, want 0 for a simulation", s.TotalCommits)
	}
}

// TestSummaryOfDistinguishesZeroFromAbsent is why HasAvg exists: a genuine average
// of zero must not be confused with a line that was never printed.
func TestSummaryOfDistinguishesZeroFromAbsent(t *testing.T) {
	absent := SummaryOf([]byte("Total commits: 0\n"))
	if absent.HasAvg {
		t.Error("HasAvg = true for output with no average line")
	}
	present := SummaryOf([]byte("Average per active day: 0\n"))
	if !present.HasAvg {
		t.Fatal("HasAvg = false for an average line reporting zero")
	}
	if present.AvgPerActiveDay != 0 {
		t.Errorf("AvgPerActiveDay = %v, want 0", present.AvgPerActiveDay)
	}
}

// TestSummaryOfIgnoresTableRows keeps a table cell from being read as a total.
//
// The commit-plan table has a "Commits (est.)" column, and a loose regex would
// match its digits. Every pattern is anchored to the start of the trimmed line to
// prevent exactly that.
func TestSummaryOfIgnoresTableRows(t *testing.T) {
	table := `┏━━━━━━━━━━━━┳━━━━━━━━━━━┳━━━━━━━━━━━━━━━━┓
┃ Date       ┃ Intensity ┃ Commits (est.) ┃
┡━━━━━━━━━━━━╇━━━━━━━━━━━╇━━━━━━━━━━━━━━━━┩
│ 2025-09-21 │     4     │              4 │
└────────────┴───────────┴────────────────┘
Total estimated commits: 128
Active days: 32
`
	s := SummaryOf([]byte(table))
	if s.TotalCommits != 128 {
		t.Errorf("TotalCommits = %d, want 128", s.TotalCommits)
	}
	if s.ActiveDays != 32 {
		t.Errorf("ActiveDays = %d, want 32", s.ActiveDays)
	}
}

// TestSummaryOfEmptyAndGarbage keeps a non-plan output from reporting figures.
func TestSummaryOfEmptyAndGarbage(t *testing.T) {
	for _, in := range []string{"", "   ", "Error: Pattern 'nope' not found", "Grid Preview\n\n███"} {
		if s := SummaryOf([]byte(in)); s.HasAny() {
			t.Errorf("SummaryOf(%q) = %+v, want nothing recovered", in, s)
		}
	}
}

// TestPatternsReadsTheCatalog pins the pattern list coming from the CLI's own
// listing rather than a Go-side constant.
func TestPatternsReadsTheCatalog(t *testing.T) {
	got := Patterns([]byte(patternCatalog))
	want := []string{"checkerboard", "heart", "wave"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Patterns = %v, want %v", got, want)
	}
}

// TestPatternsKeepsTheCLIsOrder pins that the list is not re-sorted: the CLI's
// order is its own, and reordering could change which name a caller treats as the
// default.
func TestPatternsKeepsTheCLIsOrder(t *testing.T) {
	got := Patterns([]byte("Available patterns: zeta, alpha, mu\n"))
	want := []string{"zeta", "alpha", "mu"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Patterns = %v, want %v (the CLI's order)", got, want)
	}
}

// TestPatternsEmptyWhenNothingIsAdvertised keeps an unexpected output from
// inventing names.
func TestPatternsEmptyWhenNothingIsAdvertised(t *testing.T) {
	if got := Patterns([]byte("something else entirely\n")); got != nil {
		t.Errorf("Patterns = %v, want nil", got)
	}
	if got := Patterns([]byte("Available patterns: \n")); len(got) != 0 {
		t.Errorf("Patterns = %v, want none", got)
	}
}

// TestProfilesReadsTheHelpText pins the profile list coming from the --profile
// help string, which upstream builds with available_profiles().
func TestProfilesReadsTheHelpText(t *testing.T) {
	got := Profiles([]byte(simulateHelpLine))
	want := []string{"default", "night-owl", "weekend-warrior", "9-to-5", "open-source-maintainer"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("Profiles = %v, want %v", got, want)
	}
}

// TestProfilesFallsBackRatherThanGuessing pins the failure mode.
//
// A wrapped or truncated help text must yield nothing, so the caller falls back to
// the catalogue the CLI ships. Returning the part of the list that happened to fit
// on the first line would be worse than no answer: a partial Select would hide real
// profiles and silently narrow what the user can choose.
//
// Wrapping is not expected in practice: simulate --help is invoked with COLUMNS
// pinned wide (see cli.previewColumns), so the list arrives on one line.
func TestProfilesFallsBackRatherThanGuessing(t *testing.T) {
	wrapped := "│ --profile -p TEXT  Activity profile (default, night-owl,\n│                    weekend-warrior). [default: default]"
	if got := Profiles([]byte(wrapped)); got != nil {
		t.Errorf("Profiles = %v, want nil so the caller can fall back", got)
	}
}

// TestProfilesEmptyWhenAbsent keeps a missing list from yielding junk.
func TestProfilesEmptyWhenAbsent(t *testing.T) {
	if got := Profiles([]byte("Usage: github-utils simulate [OPTIONS]\n")); got != nil {
		t.Errorf("Profiles = %v, want nil", got)
	}
}

// TestVersion pins the version parse.
func TestVersion(t *testing.T) {
	if got := Version([]byte("github-utils 2.0.0\n")); got != "2.0.0" {
		t.Errorf("Version = %q, want 2.0.0", got)
	}
	if got := Version([]byte("no version here\n")); got != "" {
		t.Errorf("Version = %q, want empty", got)
	}
}

// TestParsersTolerateAShortBuffer keeps every parser from panicking on input that
// was cut off mid-line, which is what a killed subprocess leaves behind.
func TestParsersTolerateAShortBuffer(t *testing.T) {
	for _, in := range []string{"Total estimated", "Available patterns", "Activity profile (", "github-utils"} {
		SummaryOf([]byte(in))
		Patterns([]byte(in))
		Profiles([]byte(in))
		Version([]byte(in))
	}
}

// TestDefaultProfileReadsTheAdvertisedDefault pins that the profile default comes
// from the CLI's own help text, not from list order.
func TestDefaultProfileReadsTheAdvertisedDefault(t *testing.T) {
	if got := DefaultProfile([]byte(simulateHelpLine)); got != "default" {
		t.Errorf("DefaultProfile = %q, want %q", got, "default")
	}
	// A reordered list must not change the answer.
	reordered := "│ --profile -p TEXT  Activity profile (night-owl, default, weekend-warrior). [default: default]"
	if got := DefaultProfile([]byte(reordered)); got != "default" {
		t.Errorf("DefaultProfile = %q, want default whatever the list order", got)
	}
	// A help line with no declared default yields nothing, so the caller can fall
	// back deliberately rather than guessing.
	noDefault := "│ --profile -p TEXT  Activity profile (night-owl, default)."
	if got := DefaultProfile([]byte(noDefault)); got != "" {
		t.Errorf("DefaultProfile = %q, want empty when none is advertised", got)
	}
	if got := DefaultProfile([]byte("")); got != "" {
		t.Errorf("DefaultProfile = %q, want empty for empty input", got)
	}
}
