// Package parse reads the small, machine-useful facts out of the CLI's
// human-facing output.
//
// github-utils has no --json mode: every command prints a Rich table for a
// person. Rather than re-derive the numbers in Go, this package recovers the
// summary lines and catalogues the CLI already prints, so the figures on screen
// are still the CLI's own.
package parse

import (
	"regexp"
	"strconv"
	"strings"
)

// Summary is the set of totals a plan command prints beneath its table.
//
// Field names follow the CLI's own vocabulary rather than being unified, because
// `paint` and `simulate` genuinely report different things: paint estimates
// commits for a grid, while simulate reports a plan whose second ActiveDays
// figure is the window size.
type Summary struct {
	// TotalCommits is `paint`'s "Total estimated commits: N".
	TotalCommits int
	// SimulatedCommits is `simulate`'s "Total commits: N".
	SimulatedCommits int
	// ActiveDays is "Active days: N" (paint), or the numerator of
	// "Active days: N / M" (simulate).
	ActiveDays int
	// WindowDays is the denominator of simulate's "Active days: N / M";
	// zero when the CLI printed a bare count.
	WindowDays int
	// AvgPerActiveDay is simulate's "Average per active day: X".
	AvgPerActiveDay float64
	// HasAvg records whether the average line was present, so a genuine zero is
	// distinguishable from an absent line.
	HasAvg bool
}

// HasAny reports whether any figure was recovered.
func (s Summary) HasAny() bool {
	return s.TotalCommits != 0 || s.SimulatedCommits != 0 || s.ActiveDays != 0 || s.HasAvg
}

var (
	// "Total estimated commits: 128" and "Total commits: 92". The label is
	// anchored to the start of the trimmed line so a table cell cannot match.
	reTotalEstimated = regexp.MustCompile(`^Total estimated commits:\s*(\d+)`)
	reTotalCommits   = regexp.MustCompile(`^Total commits:\s*(\d+)`)
	// "Active days: 32" or "Active days: 26 / 30".
	reActiveDays = regexp.MustCompile(`^Active days:\s*(\d+)(?:\s*/\s*(\d+))?`)
	// "Average per active day: 3.5".
	reAvgPerDay = regexp.MustCompile(`^Average per active day:\s*([0-9]+(?:\.[0-9]+)?)`)
	// The profile `simulate` pages when it is not given one, as its own help
	// advertises ("[default: default]").
	reDefaultProfile = regexp.MustCompile(`\[default:\s*([^\]]+)\]`)
	// "Available patterns: checkerboard, heart, wave".
	reAvailablePatterns = regexp.MustCompile(`Available patterns:\s*(.*)`)
	// "Activity profile (default, night-owl, weekend-warrior, 9-to-5,
	// open-source-maintainer)." from simulate --help.
	reProfileList = regexp.MustCompile(`Activity profile\s*\(([^)]*)\)`)
)

// SummaryOf recovers every figure present in a plan command's output.
//
// A missing line is left at its zero value rather than treated as an error:
// paint prints no average, and simulate prints no estimated-commit line, so
// requiring either would fail on one of them.
func SummaryOf(out []byte) Summary {
	var s Summary
	for _, raw := range strings.Split(stripANSI(string(out)), "\n") {
		line := strings.TrimSpace(raw)
		// The estimated-commit arm is tested before the plain commit one, since
		// "Total estimated commits" and "Total commits" are different labels but
		// both start with "Total".
		switch {
		case reTotalEstimated.MatchString(line):
			s.TotalCommits = firstGroup(reTotalEstimated, line)
		case reTotalCommits.MatchString(line):
			s.SimulatedCommits = firstGroup(reTotalCommits, line)
		case reActiveDays.MatchString(line):
			m := reActiveDays.FindStringSubmatch(line)
			s.ActiveDays, _ = strconv.Atoi(m[1])
			if m[2] != "" {
				s.WindowDays, _ = strconv.Atoi(m[2])
			}
		case reAvgPerDay.MatchString(line):
			value := reAvgPerDay.FindStringSubmatch(line)[1]
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				s.AvgPerActiveDay = v
				s.HasAvg = true
			}
		}
	}
	return s
}

// firstGroup returns the first capture of re as an int.
func firstGroup(re *regexp.Regexp, line string) int {
	m := re.FindStringSubmatch(line)
	if len(m) < 2 {
		return 0
	}
	v, _ := strconv.Atoi(m[1])
	return v
}

// Patterns recovers the built-in pattern names from a bare `paint` invocation.
//
// There is no listing subcommand. Invoked with no input source, paint prints
// "Available patterns: checkerboard, heart, wave" and exits 1; the caller is
// expected to read stdout even on that exit.
func Patterns(out []byte) []string {
	for _, raw := range strings.Split(stripANSI(string(out)), "\n") {
		m := reAvailablePatterns.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		return splitList(m[1])
	}
	return nil
}

// Profiles recovers the activity-profile names from `simulate --help`.
// The --profile help string upstream is
// f"Activity profile ({', '.join(available_profiles())})", so the help text
// advertises the real set. Reading it here means a profile added in Python shows
// up in the Go front-end without a Go change.
//
// The match is deliberately confined to one line. The caller pins COLUMNS wide
// when it asks for this help, so the list is not wrapped; if the help text ever
// stops matching, this returns nothing and the caller falls back to the catalogue
// the CLI ships. Returning a truncated list would be worse than no list, because a
// partial Select hides real options.
func Profiles(out []byte) []string {
	for _, raw := range strings.Split(stripANSI(string(out)), "\n") {
		m := reProfileList.FindStringSubmatch(raw)
		if m == nil {
			continue
		}
		return splitList(m[1])
	}
	return nil
}

// Version recovers the version from `github-utils --version` ("github-utils
// 2.0.0").
func Version(out []byte) string {
	for _, raw := range strings.Split(stripANSI(string(out)), "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "github-utils ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "github-utils "))
		}
	}
	return ""
}

// DefaultProfile recovers the profile simulate pages when it is not told one,
// from the "[default: X]" suffix on the --profile help line.
//
// Reading it keeps a simulate run in this front-end on the same profile the Python
// CLI would pick. Assuming the first name in the list is the default would be a
// coincidence, not a contract.
func DefaultProfile(out []byte) string {
	for _, raw := range strings.Split(stripANSI(string(out)), "\n") {
		if !strings.Contains(raw, "Activity profile") {
			continue
		}
		if m := reDefaultProfile.FindStringSubmatch(raw); m != nil {
			return strings.TrimSpace(m[1])
		}
	}
	return ""
}

// splitList splits a comma-separated catalogue, dropping empties and padding.
//
// The order is the CLI's own, deliberately: simulate's default is the first name
// Python yields, so re-sorting here could make the front-end's default differ
// from the Python CLI's.
func splitList(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// stripANSI removes SGR colour sequences, which appear only when something has
// forced colour on a pipe.
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
