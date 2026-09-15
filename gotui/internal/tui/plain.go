package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jasperan/github-utils/gotui/internal/cli"
	"github.com/jasperan/github-utils/gotui/internal/grid"
	"github.com/jasperan/github-utils/gotui/internal/parse"
)

// Plain actions, which name a fully-specified run rather than a form.
const (
	PlainPaintText    = "paint-text"
	PlainPaintPattern = "paint-pattern"
	PlainSimulate     = "simulate"
	PlainCompare      = "compare"
	PlainHelp         = "help"
)

// PlainAction is a run that needs no prompt.
type PlainAction struct {
	Kind      string
	Text      string
	Pattern   string
	Intensity int
	Days      int
	Profile   string
	Seed      *int
	Before    string
	After     string
}

// ErrNoPlainAction is returned when the flags name no action to run.
var ErrNoPlainAction = errors.New("no action was named on the command line")

// ResolvePlain derives the action from flags.
//
// A run is non-interactive whenever stdin is not a terminal, or --plain was
// passed, or ACCESSIBLE is set. In that case a form can never be answered, so a
// pipe or a script must be able to name the whole action up front. When more than
// one action is named the first in this order wins, and that order is a fixed
// contract rather than a guess.
func ResolvePlain(seed Seed) (PlainAction, error) {
	intensity, err := resolveIntensity(seed.Intensity)
	if err != nil {
		return PlainAction{}, err
	}

	switch {
	case strings.TrimSpace(seed.Text) != "":
		text := strings.TrimSpace(seed.Text)
		if err := ValidatePaintText(text); err != nil {
			return PlainAction{}, err
		}
		return PlainAction{Kind: PlainPaintText, Text: text, Intensity: intensity}, nil
	case strings.TrimSpace(seed.Pattern) != "":
		return PlainAction{Kind: PlainPaintPattern, Pattern: strings.TrimSpace(seed.Pattern)}, nil
	case seed.Simulate:
		// Only a value the user actually typed is validated; an absent --days means
		// the CLI's own default, not a missing required field. Validating the raw
		// empty string rejected `--simulate` on its own for a flag never passed.
		days, err := resolveDays(seed.Days)
		if err != nil {
			return PlainAction{}, err
		}
		if err := ValidateOptionalSeed(seed.Seed); err != nil {
			return PlainAction{}, err
		}
		var seedValue *int
		if v := strings.TrimSpace(seed.Seed); v != "" {
			parsed, perr := strconv.Atoi(v)
			if perr != nil {
				return PlainAction{}, perr
			}
			seedValue = &parsed
		}
		return PlainAction{
			Kind:    PlainSimulate,
			Days:    days,
			Profile: strings.TrimSpace(seed.Profile),
			Seed:    seedValue,
		}, nil
	case strings.TrimSpace(seed.Before) != "" || strings.TrimSpace(seed.After) != "":
		if err := ValidateSnapshotPath(seed.Before); err != nil {
			return PlainAction{}, err
		}
		if err := ValidateSnapshotPath(seed.After); err != nil {
			return PlainAction{}, err
		}
		return PlainAction{
			Kind:   PlainCompare,
			Before: strings.TrimSpace(seed.Before),
			After:  strings.TrimSpace(seed.After),
		}, nil
	default:
		return PlainAction{Kind: PlainHelp}, ErrNoPlainAction
	}
}

// resolveDays validates a typed --days, defaulting only when it was omitted.
//
// A blank value is the CLI's own default; a typo is an error. Falling back to the
// default on unparseable input would silently run a different window than the user
// asked for.
func resolveDays(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DefaultDays, nil
	}
	if err := ValidateDays(trimmed); err != nil {
		return 0, err
	}
	return strconv.Atoi(trimmed)
}

// resolveIntensity validates a typed --intensity, defaulting only when omitted.
func resolveIntensity(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return MaxIntensity, nil
	}
	if err := ValidateIntensity(trimmed); err != nil {
		return 0, err
	}
	return strconv.Atoi(trimmed)
}

// RunPlain runs a resolved action and returns unstyled text.
//
// It is the screen-reader and pipeline path: nothing here depends on a terminal,
// and no form is ever constructed, because an embedded huh form ignores
// WithAccessible (see the note in the canonical theme) and would hang on a pipe.
func RunPlain(opts Options, action PlainAction) (string, error) {
	switch action.Kind {
	case PlainPaintText:
		return runPlainPaint(opts, cli.TextPreview(opts.Launcher, action.Text, action.Intensity),
			cli.TextPlan(opts.Launcher, action.Text, action.Intensity))
	case PlainPaintPattern:
		return runPlainPaint(opts, cli.PatternPreview(opts.Launcher, action.Pattern),
			cli.PatternPlan(opts.Launcher, action.Pattern))
	case PlainSimulate:
		inv, err := cli.SimulatePlan(opts.Launcher, cli.SimulateOptions{
			Days:    action.Days,
			Profile: action.Profile,
			Seed:    action.Seed,
			DryRun:  true,
		})
		if err != nil {
			return "", err
		}
		out, err := runOne(opts, inv)
		return plainResult(inv.CommandLine(), out, err), err
	case PlainCompare:
		inv, err := cli.Compare(opts.Launcher, cli.CompareOptions{
			Before: action.Before,
			After:  action.After,
		})
		if err != nil {
			return "", err
		}
		out, err := runOne(opts, inv)
		return plainResult(inv.CommandLine(), out, err), err
	default:
		return PlainUsage(opts), nil
	}
}

// runPlainPaint runs the preview and the plan, preferring the preview's outcome.
func runPlainPaint(opts Options, preview, plan cli.Invocation) (string, error) {
	gridOut, gridErr := runOne(opts, preview)
	planOut, planErr := runOne(opts, plan)

	report := plainResult(preview.CommandLine(), planOut, planErr)
	if parsed, err := grid.ParsePreview(gridOut); err == nil {
		report = plainPreviewBlock(parsed) + "\n" + report
	}

	// The preview is the result the user asked for, so its failure is the one
	// reported; a plan failure after a good grid is still returned so the exit
	// status is honest.
	if gridErr != nil {
		return report, gridErr
	}
	return report, planErr
}

// runOne executes one invocation, returning whatever output it produced.
func runOne(opts Options, inv cli.Invocation) ([]byte, error) {
	if opts.Runner == nil {
		return nil, errNoRunner
	}
	return opts.Runner.Run(inv, opts.Timeout)
}

// plainPreviewBlock renders a grid with the block glyphs and no styling.
func plainPreviewBlock(g *grid.Grid) string {
	var b strings.Builder
	for row := 0; row < grid.Rows; row++ {
		for col := 0; col < grid.Cols; col++ {
			b.WriteRune(heatGlyph[g.At(row, col)])
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

// plainResult composes the unstyled report: the command, the recovered totals,
// and the CLI's own output.
//
// The CLI's output is passed through unchanged, because in plain mode a caller may
// be parsing it; the totals are repeated above it only when they were found.
func plainResult(command string, out []byte, err error) string {
	var b strings.Builder
	b.WriteString("$ " + command + "\n")
	if summary := parse.SummaryOf(out); summary.HasAny() {
		if p := plainSummary(summary); p != "" {
			b.WriteString(p + "\n")
		}
	}
	if text := strings.TrimSpace(string(out)); text != "" {
		b.WriteString(text + "\n")
	}
	if err != nil {
		b.WriteString("error: " + err.Error() + "\n")
	}
	return b.String()
}

// plainSummary renders the recovered totals without styling.
func plainSummary(s parse.Summary) string {
	var parts []string
	if s.TotalCommits > 0 {
		parts = append(parts, "total estimated commits: "+strconv.Itoa(s.TotalCommits))
	}
	if s.SimulatedCommits > 0 {
		parts = append(parts, "total commits: "+strconv.Itoa(s.SimulatedCommits))
	}
	if s.WindowDays > 0 {
		parts = append(parts, "active days: "+strconv.Itoa(s.ActiveDays)+" / "+strconv.Itoa(s.WindowDays))
	} else if s.ActiveDays > 0 {
		parts = append(parts, "active days: "+strconv.Itoa(s.ActiveDays))
	}
	if s.HasAvg {
		parts = append(parts, "average per active day: "+strconv.FormatFloat(s.AvgPerActiveDay, 'f', -1, 64))
	}
	return strings.Join(parts, "\n")
}

// PlainUsage is the text shown when no action was named and no terminal is
// available to ask on.
//
// It names the flags that would make the run work, which is the difference
// between an actionable refusal and a blank screen.
func PlainUsage(opts Options) string {
	var b strings.Builder
	b.WriteString("github-utils-tui — an additional way to run github-utils\n\n")
	b.WriteString("No terminal is available on stdin, so nothing can be prompted for.\n")
	b.WriteString("Name the action on the command line instead:\n\n")
	b.WriteString("  github-utils-tui --paint \"HELLO\" [--intensity 1-4]\n")
	b.WriteString("  github-utils-tui --pattern wave\n")
	b.WriteString("  github-utils-tui --simulate [--days N] [--profile P] [--seed S]\n")
	b.WriteString("  github-utils-tui --before A.json --after B.json\n\n")
	b.WriteString("Run on a terminal for the interactive UI, or pass --plain to force this mode.\n\n")
	b.WriteString(fmt.Sprintf("delegates to: %s\n", opts.Launcher.Describe("--help")))
	if len(opts.Patterns) > 0 {
		b.WriteString("patterns: " + strings.Join(opts.Patterns, ", ") + "\n")
	}
	if len(opts.Profiles) > 0 {
		b.WriteString("profiles: " + strings.Join(opts.Profiles, ", ") + "\n")
	}
	return b.String()
}
