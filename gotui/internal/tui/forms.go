package tui

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"charm.land/huh/v2"

	"github.com/jasperan/github-utils/gotui/internal/grid"
	"github.com/jasperan/github-utils/gotui/internal/huhstyle"
)

// Action identifiers. They are the values a Select carries, so they are stored
// as plain strings rather than compared against a label the user might reword.
const (
	ActionPaintText    = "paint-text"
	ActionPaintPattern = "paint-pattern"
	ActionSimulate     = "simulate"
	ActionCompare      = "compare"
	ActionAbout        = "about"
)

// maxTextChars is how many characters fit on a painted grid.
//
// Measured, not assumed. font.py lays each glyph out in 5 columns with a 1-column
// gap, and drops anything past column 51, so 8 characters occupy 8*5+7 = 47
// columns and 9 would need 53. Driving the CLI confirms it: painting 8, 9 and 10
// identical letters all report the same "Total estimated commits: 544" and
// "Active days: 136", so the ninth is discarded silently.
//
// Because that drop is silent, the form refuses a longer string instead of
// letting the user believe all of it was painted.
const maxTextChars = grid.Cols / 6

// DefaultDays mirrors the CLI's own --days default, so the front-end does not
// quietly invent a different window.
const DefaultDays = 90

// MinIntensity and MaxIntensity are the bounds paint declares for --intensity.
const (
	MinIntensity = 1
	MaxIntensity = 4
)

// themed applies the project-wide token theme to a form.
//
// WithAccessible is set for intent, but note that huh v2.0.3 only consults it
// inside Form.RunWithContext: an embedded form renders identically either way, so
// the screen-reader path in this front-end is the plain renderer, not this form.
func themed(f *huh.Form) *huh.Form {
	return f.WithTheme(huh.ThemeFunc(huhstyle.Theme)).WithAccessible(huhstyle.Accessible())
}

// MenuAnswers holds the launcher selection.
type MenuAnswers struct {
	Choice string
}

// DefaultChoice is the launcher's initial selection.
const DefaultChoice = ActionPaintText

// MenuForm builds the launcher.
//
// version is shown in the option descriptions so the user can confirm which CLI
// this front-end will delegate to.
func MenuForm(ans *MenuAnswers, version string) *huh.Form {
	if ans.Choice == "" {
		ans.Choice = DefaultChoice
	}
	cliHint := "runs the repo's own `github-utils` CLI"
	if strings.TrimSpace(version) != "" {
		cliHint = "delegates to github-utils " + strings.TrimSpace(version)
	}
	return themed(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("What do you want to do?").
			Description(cliHint).
			Options(
				huh.NewOption("Paint text on the graph", ActionPaintText),
				huh.NewOption("Paint a built-in pattern", ActionPaintPattern),
				huh.NewOption("Simulate activity", ActionSimulate),
				huh.NewOption("Compare two snapshots", ActionCompare),
				huh.NewOption("About this front-end", ActionAbout),
			).
			Value(&ans.Choice),
	)))
}

// PaintTextAnswers holds the text-painting form.
type PaintTextAnswers struct {
	Text      string
	Intensity string
}

// IntensityOptions are the levels paint accepts, labelled for a reader.
var IntensityOptions = []struct {
	Label string
	Value string
}{
	{"1 — light", "1"},
	{"2 — medium", "2"},
	{"3 — heavy", "3"},
	{"4 — solid (default)", "4"},
}

// DefaultIntensityValue is the CLI's own default for --intensity.
const DefaultIntensityValue = "4"

// PaintTextForm builds the text form.
//
// Intensity is a Select rather than a typed number: the CLI bounds it to 1-4, and
// a fixed set cannot be typed out of range.
func PaintTextForm(ans *PaintTextAnswers) *huh.Form {
	options := make([]huh.Option[string], 0, len(IntensityOptions))
	for _, o := range IntensityOptions {
		options = append(options, huh.NewOption(o.Label, o.Value))
	}
	// Seed only from an option that exists. A huh Select whose value is not among
	// its options silently falls back to the FIRST one, which would paint at a
	// different intensity than the caller asked for.
	// Seed the intensity, distinguishing three cases. An EMPTY value means "no
	// preference", which is the CLI's own default (4) -- not the first option in the
	// list, which is 1. A value that arrived from a flag but is not among the options
	// is replaced by the CLI's default too. huh has already written the first option
	// into the bound field by this point, so the rejection is detected first.
	if strings.TrimSpace(ans.Intensity) == "" || rejectedValue(ans.Intensity, options) != "" {
		ans.Intensity = DefaultIntensityValue
	}
	ans.Intensity = selectableValue(ans.Intensity, options)
	if strings.TrimSpace(ans.Intensity) == "" {
		ans.Intensity = DefaultIntensityValue
	}

	return themed(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Text to paint").
			Description(fmt.Sprintf("up to %d characters; each occupies 5 columns. Painted in upper case.", maxTextChars)).
			Placeholder("HELLO").
			CharLimit(maxTextChars*4).
			Validate(ValidatePaintText).
			Value(&ans.Text),
		huh.NewSelect[string]().
			Title("Intensity").
			Description("how dark each painted day is").
			Options(options...).
			Value(&ans.Intensity),
	).Title("Paint text")))
}

// PaintPatternAnswers holds the pattern form.
type PaintPatternAnswers struct {
	Pattern string
	// Unavailable records a requested pattern that the CLI does not offer.
	//
	// It exists because a huh Select cannot be prevented from substituting the first
	// option: huh.NewSelect().Value(&v) writes the selection back IMMEDIATELY at
	// construction, so binding a value that is not among the options silently
	// replaces it. The request is therefore remembered here and refused by the
	// command builder, rather than being quietly swapped for a different pattern.
	Unavailable string
}

// PaintPatternForm builds the pattern form from the names the CLI advertised.
//
// patterns comes from parse.Patterns over a bare `paint` run. An empty list is
// reported by the command builder rather than shown as an empty Select, because a
// Select with no options can never be completed.
func PaintPatternForm(ans *PaintPatternAnswers, patterns []string) *huh.Form {
	options := make([]huh.Option[string], 0, len(patterns))
	for _, name := range patterns {
		options = append(options, huh.NewOption(name, name))
	}
	if len(options) == 0 {
		options = append(options, huh.NewOption("no patterns available", ""))
	}
	ans.Unavailable = rejectedValue(ans.Pattern, options)
	ans.Pattern = selectableValue(ans.Pattern, options)

	return themed(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title("Pattern").
			Description("the built-in patterns this CLI ships").
			Options(options...).
			Value(&ans.Pattern),
	).Title("Paint pattern")))
}

// SimulateAnswers holds the simulation form.
type SimulateAnswers struct {
	Days    string
	Profile string
	Seed    string
	// Unavailable records a requested profile the CLI does not offer; see
	// PaintPatternAnswers.Unavailable for why a Select cannot simply refuse one.
	Unavailable string
}

// SimulateForm builds the simulation form.
//
// profiles and defaultProfile come from the --profile help text, so the list and the
// default are the CLI's own. The seed is optional and left blank for a random plan.
func SimulateForm(ans *SimulateAnswers, profiles []string, defaultProfile string) *huh.Form {
	if strings.TrimSpace(ans.Days) == "" {
		ans.Days = strconv.Itoa(DefaultDays)
	}
	options := make([]huh.Option[string], 0, len(profiles))
	for _, name := range profiles {
		options = append(options, huh.NewOption(name, name))
	}
	if len(options) == 0 {
		options = append(options, huh.NewOption("no profiles available", ""))
	}

	requested := strings.TrimSpace(ans.Profile)
	// A name the CLI does not offer is recorded and refused; see
	// PaintPatternAnswers.Unavailable for why a Select cannot simply reject one.
	ans.Unavailable = rejectedValue(requested, options)
	if ans.Unavailable != "" {
		requested = ""
	}
	if requested == "" {
		// Nothing was asked for, so start from the profile the CLI itself would use
		// rather than whichever name happens to be listed first. They agree today;
		// this keeps them agreeing if the Python-side list is reordered.
		requested = strings.TrimSpace(defaultProfile)
	}
	ans.Profile = selectableValue(requested, options)

	return themed(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Days to simulate").
			Description("the CLI requires at least 1").
			Placeholder(strconv.Itoa(DefaultDays)).
			CharLimit(5).
			Validate(ValidateDays).
			Value(&ans.Days),
		huh.NewSelect[string]().
			Title("Activity profile").
			Description("a rhythm from the CLI's own profile set").
			Options(options...).
			Value(&ans.Profile),
		huh.NewInput().
			Title("Seed (optional)").
			Description("leave blank for a random plan; the same seed reproduces a plan exactly").
			Placeholder("random").
			CharLimit(10).
			Validate(ValidateOptionalSeed).
			Value(&ans.Seed),
	).Title("Simulate activity")))
}

// CompareAnswers holds the snapshot-comparison form.
type CompareAnswers struct {
	Before string
	After  string
}

// CompareForm builds the comparison form.
//
// Both paths are required by the CLI, so both are validated as existing files
// here rather than letting the CLI report it after a round trip.
func CompareForm(ans *CompareAnswers) *huh.Form {
	return themed(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title("Before snapshot").
			Description("path to a snapshot JSON file").
			Placeholder("snap-before.json").
			Validate(ValidateSnapshotPath).
			Value(&ans.Before),
		huh.NewInput().
			Title("After snapshot").
			Description("path to a snapshot JSON file").
			Placeholder("snap-after.json").
			Validate(ValidateSnapshotPath).
			Value(&ans.After),
	).Title("Compare snapshots")))
}

// ValidatePaintText requires one to maxTextChars characters.
func ValidatePaintText(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("enter some text to paint")
	}
	if n := len([]rune(trimmed)); n > maxTextChars {
		return fmt.Errorf("%d characters is more than the grid fits; the CLI would drop the rest after %d", n, maxTextChars)
	}
	return nil
}

// ValidateDays requires a whole number of at least 1, which is the CLI's own
// bound on --days.
func ValidateDays(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("enter a number of days")
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		return errors.New("days must be a whole number")
	}
	if n < 1 {
		return errors.New("the CLI requires at least 1 day")
	}
	return nil
}

// ValidateIntensity bounds a typed intensity to the range paint declares.
//
// The form uses a Select, so it cannot go out of range there; this exists for the
// flag path, where the value is typed.
func ValidateIntensity(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("enter an intensity")
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil {
		return errors.New("intensity must be a whole number")
	}
	if n < MinIntensity || n > MaxIntensity {
		return fmt.Errorf("intensity must be between %d and %d", MinIntensity, MaxIntensity)
	}
	return nil
}

// ValidateOptionalSeed accepts an empty value (random plan) or a whole number.
func ValidateOptionalSeed(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	if _, err := strconv.Atoi(trimmed); err != nil {
		return errors.New("the seed must be a whole number, or blank for a random plan")
	}
	return nil
}

// ValidateSnapshotPath requires a path that exists, since the CLI's compare
// command exits non-zero on a missing file.
func ValidateSnapshotPath(value string) error {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return errors.New("enter a snapshot file path")
	}
	info, err := os.Stat(trimmed)
	if err != nil {
		return fmt.Errorf("no such file: %s", trimmed)
	}
	if info.IsDir() {
		return errors.New("that is a directory, not a snapshot file")
	}
	return nil
}

// selectableValue returns the value a Select should be bound to.
//
//   - the value is available -> keep it
//   - nothing was requested  -> default to the first real option
//   - anything else          -> the first real option, with the rejected request
//     recorded by rejectedValue so it is never silently substituted
func selectableValue(requested string, options []huh.Option[string]) string {
	for _, o := range options {
		if o.Value == requested {
			return requested
		}
	}
	if len(options) > 0 {
		return options[0].Value
	}
	return ""
}

// rejectedValue reports a requested value that the options do not contain.
//
// This is the only way to detect the substitution: huh has already overwritten the
// bound value with the first option by the time the form is built, so the original
// request has to be captured before it is bound. An empty request is not a
// rejection -- it is simply no preference.
func rejectedValue(requested string, options []huh.Option[string]) string {
	trimmed := strings.TrimSpace(requested)
	if trimmed == "" {
		return ""
	}
	for _, o := range options {
		if o.Value == trimmed {
			return ""
		}
	}
	return trimmed
}
