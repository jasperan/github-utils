// Command github-utils-tui is a Go front-end for the github-utils Python CLI.
//
// It is an ADDITIONAL way to run the same toolkit, not a replacement: every glyph,
// pattern, activity profile and commit estimate it shows was produced by running
// the installed `github-utils` binary. Nothing about painting, the pixel font, the
// pattern loader or the profile engines is reimplemented here, so a Go user and a
// Python user get identical results.
//
// Two paths:
//   - a terminal: a huh-driven menu (paint text, paint a pattern, simulate
//     activity, compare snapshots) with the grid rendered as a token-coloured
//     calendar above the CLI's own plan table
//   - no terminal (a pipe, a script, --plain, or ACCESSIBLE): flags only, never a
//     prompt
//
// Everything this front-end asks the CLI for is read-only. paint and simulate are
// driven with --preview and --dry-run, and their execute branches are unimplemented
// upstream ("Execution not yet implemented"), so no run here can push or commit.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/jasperan/github-utils/gotui/internal/cli"
	"github.com/jasperan/github-utils/gotui/internal/huhstyle"
	"github.com/jasperan/github-utils/gotui/internal/parse"
	"github.com/jasperan/github-utils/gotui/internal/run"
	"github.com/jasperan/github-utils/gotui/internal/tui"
)

// version is the front-end's own version, not the CLI's.
const version = "0.1.0"

func main() {
	os.Exit(realMain())
}

func realMain() int {
	var (
		binary      = flag.String("bin", "", "path to the github-utils CLI (default: discover it)")
		timeout     = flag.Duration("timeout", tui.DefaultTimeout, "per-call subprocess timeout")
		plain       = flag.Bool("plain", false, "never open the full-screen UI")
		showVersion = flag.Bool("version", false, "print the front-end version, then exit")
		list        = flag.Bool("list", false, "print the CLI's patterns and profiles, then exit")

		text      = flag.String("paint", "", "text to paint (up to 8 characters)")
		pattern   = flag.String("pattern", "", "built-in pattern name, e.g. wave")
		intensity = flag.String("intensity", "", "paint intensity, 1-4 (default 4)")
		simulate  = flag.Bool("simulate", false, "simulate activity")
		days      = flag.String("days", "", "days to simulate for -simulate (default 90)")
		profile   = flag.String("profile", "", "activity profile for -simulate")
		seed      = flag.String("seed", "", "random seed for -simulate; blank is random")
		before    = flag.String("before", "", "before snapshot JSON for a comparison")
		after     = flag.String("after", "", "after snapshot JSON for a comparison")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("github-utils-tui %s\n", version)
		return 0
	}

	launcher, err := cli.Resolve(*binary)
	if err != nil {
		fmt.Fprintf(os.Stderr, "github-utils-tui: %v\n", err)
		return 2
	}

	// The catalogues and the version are read from the CLI itself, so a pattern or
	// profile added on the Python side appears here without a Go change. A probe
	// failure is not fatal: the forms fall back to the values the CLI ships today
	// and the UI still works, so it is reported on stderr rather than refusing to
	// start.
	probe := probeCLI(launcher, *timeout)

	if *list {
		fmt.Println("patterns: " + strings.Join(probe.patterns, ", "))
		fmt.Println("profiles: " + strings.Join(probe.profiles, ", "))
		if probe.version != "" {
			fmt.Println("version: " + probe.version)
		}
		return 0
	}

	opts := tui.Options{
		Launcher:       launcher,
		Runner:         run.ShellExec{},
		Timeout:        *timeout,
		Version:        probe.version,
		Patterns:       probe.patterns,
		Profiles:       probe.profiles,
		DefaultProfile: probe.defaultProfile,
		Seed: tui.Seed{
			Text:      *text,
			Pattern:   *pattern,
			Intensity: *intensity,
			Simulate:  *simulate,
			Days:      *days,
			Profile:   *profile,
			Seed:      *seed,
			Before:    *before,
			After:     *after,
		},
	}

	// --- screen-reader path ------------------------------------------------------
	// huh v2 consults WithAccessible only in its standalone Run path, so an embedded
	// form is not accessible and this front-end does not pretend otherwise: the plain
	// renderer serves this path instead.
	if huhstyle.Accessible() {
		return runPlain(opts)
	}

	// --- non-interactive path ----------------------------------------------------
	if !huhstyle.Interactive() || *plain {
		return runPlain(opts)
	}

	// --- full-screen path --------------------------------------------------------
	model := tui.New(opts)
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintf(os.Stderr, "github-utils-tui: %v\n", err)
		return 1
	}
	return 0
}

// catalogue is what the startup probe learned from the CLI.
type catalogue struct {
	version        string
	patterns       []string
	profiles       []string
	defaultProfile string
}

// probeCLI asks the CLI for its version and its catalogues.
//
// Each probe is independent and best-effort: a checkout missing one capability
// should degrade to the other, not fail the whole start.
func probeCLI(launcher cli.Launcher, timeout time.Duration) catalogue {
	var out catalogue
	exec := run.ShellExec{}

	if raw, err := exec.Run(cli.Version(launcher), timeout); err == nil {
		out.version = parse.Version(raw)
	}
	// A bare `paint` exits 1 on purpose while printing the pattern catalogue, so a
	// non-nil error here is expected and the output is still read.
	if raw, _ := exec.Run(cli.PatternCatalog(launcher), timeout); len(raw) > 0 {
		out.patterns = parse.Patterns(raw)
	}
	if raw, err := exec.Run(cli.SimulateHelp(launcher), timeout); err == nil {
		out.profiles = parse.Profiles(raw)
		out.defaultProfile = parse.DefaultProfile(raw)
	}
	return out
}

// runPlain resolves flags with no terminal at all.
//
// A run that names no action prints the usage rather than exiting silently, so a
// mistyped invocation is distinguishable from a working command. A run that names
// an action but gets it wrong reports the specific error instead: showing the
// usage for "that text is too long" would bury the actual problem.
func runPlain(opts tui.Options) int {
	action, err := tui.ResolvePlain(opts.Seed)
	switch {
	case errors.Is(err, tui.ErrNoPlainAction):
		fmt.Fprint(os.Stdout, tui.PlainUsage(opts))
		return 2
	case err != nil:
		fmt.Fprintf(os.Stderr, "github-utils-tui: %v\n", err)
		return 2
	}
	report, err := tui.RunPlain(opts, action)
	fmt.Fprint(os.Stdout, report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "github-utils-tui: %v\n", err)
		return 1
	}
	return 0
}
