package cli

import (
	"errors"
	"strconv"
	"strings"
)

// previewColumns is the terminal width handed to Rich for grid previews.
//
// Rich honours COLUMNS even when stdout is not a terminal, and it otherwise
// falls back to 80 columns. A user with a narrow COLUMNS exported would make
// Rich wrap each 52-column grid row onto two lines, which would silently break
// the parse, so preview calls pin a width wide enough for any grid.
const previewColumns = "200"

// previewEnv is the child environment for a grid-preview call.
func previewEnv() []string {
	return EnvWith(map[string]string{"COLUMNS": previewColumns})
}

// plainEnv is the child environment for calls whose Rich table we display
// verbatim. Widening it keeps a wide table (the commit plan is three columns)
// from wrapping mid-row, which would make the captured output unreadable.
func plainEnv() []string {
	return EnvWith(map[string]string{"COLUMNS": previewColumns})
}

// Version builds `github-utils --version`.
//
// The CLI prints "github-utils <version>", and exit code 0.
func Version(l Launcher) Invocation {
	return Invocation{Argv: l.Argv("--version"), Env: EnvWith(nil), Dir: l.Dir}
}

// TextPreview builds `github-utils paint <TEXT> --preview --intensity N`.
//
// --preview is used on its own (no --dry-run) so the output is exactly the grid
// and nothing else; paint prints the commit plan only under --dry-run.
func TextPreview(l Launcher, text string, intensity int) Invocation {
	args := []string{"paint", text, "--preview"}
	if intensity > 0 {
		args = append(args, "--intensity", strconv.Itoa(intensity))
	}
	return Invocation{Argv: l.Argv(args...), Env: previewEnv(), Dir: l.Dir}
}

// PatternPreview builds `github-utils paint --pattern <NAME> --preview`.
func PatternPreview(l Launcher, pattern string) Invocation {
	return Invocation{
		Argv: l.Argv("paint", "--pattern", pattern, "--preview"),
		Env:  previewEnv(),
		Dir:  l.Dir,
	}
}

// FilePreview builds `github-utils paint --file <PATH> --preview`.
//
// paint takes exactly one of a text argument, --pattern or --file, and rejects a
// combination, so only one of these builders is ever used for a given run.
func FilePreview(l Launcher, path string) Invocation {
	return Invocation{
		Argv: l.Argv("paint", "--file", path, "--preview"),
		Env:  previewEnv(),
		Dir:  l.Dir,
	}
}

// TextPlan builds `github-utils paint <TEXT> --dry-run --intensity N`, which
// prints the commit-plan table plus the "Total estimated commits" and
// "Active days" summary lines.
func TextPlan(l Launcher, text string, intensity int) Invocation {
	args := []string{"paint", text, "--dry-run"}
	if intensity > 0 {
		args = append(args, "--intensity", strconv.Itoa(intensity))
	}
	return Invocation{Argv: l.Argv(args...), Env: plainEnv(), Dir: l.Dir}
}

// PatternPlan builds `github-utils paint --pattern <NAME> --dry-run`.
func PatternPlan(l Launcher, pattern string) Invocation {
	return Invocation{
		Argv: l.Argv("paint", "--pattern", pattern, "--dry-run"),
		Env:  plainEnv(),
		Dir:  l.Dir,
	}
}

// PatternCatalog builds a bare `github-utils paint`, which is how the CLI
// advertises the built-in pattern set.
//
// paint requires one of a text argument, --pattern or --file; invoked with none
// it prints "Error: Provide text argument, --pattern, or --file." followed by
// "Available patterns: checkerboard, heart, wave" and exits 1. There is no
// dedicated listing subcommand, so this is the CLI's own authoritative list, and
// the runner reads stdout even on a non-zero exit.
func PatternCatalog(l Launcher) Invocation {
	return Invocation{Argv: l.Argv("paint"), Env: plainEnv(), Dir: l.Dir}
}

// SimulateOptions mirrors the top-level options of `github-utils simulate`.
type SimulateOptions struct {
	// Days is the simulated window; the CLI declares a minimum of 1.
	Days int
	// Profile is an activity profile name; empty leaves the CLI's own default
	// ("default") in place rather than pinning one here.
	Profile string
	// Seed makes the plan reproducible. A nil seed omits --seed, which is what
	// asks for a random plan; zero is a valid seed and is passed through.
	Seed *int
	// DryRun is always set by the front-end: it shows the plan without
	// executing, and the CLI refuses to execute anyway ("Execution not yet
	// implemented"), so this is the only meaningful mode today.
	DryRun bool
}

// SimulatePlan builds `github-utils simulate --days N [--profile P] [--seed S] --dry-run`.
func SimulatePlan(l Launcher, opts SimulateOptions) (Invocation, error) {
	if opts.Days < 1 {
		return Invocation{}, errors.New("simulate needs at least one day")
	}
	args := []string{"simulate", "--days", strconv.Itoa(opts.Days)}
	if p := strings.TrimSpace(opts.Profile); p != "" {
		args = append(args, "--profile", p)
	}
	if opts.Seed != nil {
		args = append(args, "--seed", strconv.Itoa(*opts.Seed))
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}
	return Invocation{Argv: l.Argv(args...), Env: plainEnv(), Dir: l.Dir}, nil
}

// SimulateHelp builds `github-utils simulate --help`.
//
// This is how the front-end learns the profile set: the --profile help string is
// built with `', '.join(available_profiles())`, so it advertises the real,
// current names. Read from the CLI rather than hard-coded, so a profile added to
// the Python side appears here without a Go change.
func SimulateHelp(l Launcher) Invocation {
	return Invocation{Argv: l.Argv("simulate", "--help"), Env: plainEnv(), Dir: l.Dir}
}

// Snapshot builds `github-utils stats snapshot`, which reports the credentials it
// needs instead of doing anything without them. Read-only either way.
func Snapshot(l Launcher) Invocation {
	return Invocation{Argv: l.Argv("stats", "snapshot"), Env: plainEnv(), Dir: l.Dir}
}

// CompareOptions mirrors `github-utils stats compare`.
type CompareOptions struct {
	// Before and After are paths to snapshot JSON files; both are required by the
	// CLI.
	Before string
	After  string
}

// Compare builds `github-utils stats compare --before B --after A`.
func Compare(l Launcher, opts CompareOptions) (Invocation, error) {
	b := strings.TrimSpace(opts.Before)
	a := strings.TrimSpace(opts.After)
	if b == "" || a == "" {
		return Invocation{}, errors.New("comparing snapshots needs both a before and an after file")
	}
	return Invocation{
		Argv: l.Argv("stats", "compare", "--before", b, "--after", a),
		Env:  plainEnv(),
		Dir:  l.Dir,
	}, nil
}
