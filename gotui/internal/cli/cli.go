// Package cli locates the real github-utils CLI and builds the argv/env for
// invoking it.
//
// Every subcommand and flag spelling here is taken from
// src/github_utils/cli.py and its Typer command modules, and cross-checked
// against `github-utils <cmd> --help` on a working checkout. Nothing is guessed.
//
// The front-end never reimplements a glyph, a pattern, a profile or a commit
// estimate: it runs this CLI and renders what comes back, so a Go user and a
// Python user get identical numbers.
package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// BinaryName is the console script declared in pyproject.toml
// ([project.scripts] github-utils = "github_utils.cli:app").
const BinaryName = "github-utils"

// EntryPoint is that console-script target. It identifies the project when this
// front-end has to find the checkout on its own.
const EntryPoint = "github_utils.cli:app"

// EnvBinary overrides CLI discovery, for a checkout that is not on PATH.
const EnvBinary = "GITHUB_UTILS_BIN"

// ErrBinaryNotFound is returned when the github-utils CLI cannot be located.
//
// This front-end cannot work without it, so the error names the fix rather than
// failing later with an opaque exec error.
var ErrBinaryNotFound = errors.New(
	"the github-utils CLI was not found; this front-end delegates to it rather than " +
		"reimplementing it, so install it (uv sync --extra dev, then use .venv/bin/github-utils, " +
		"or pip install github-utils) or point " + EnvBinary + " at its entry point",
)

// Launcher is how the CLI is reached: a program, an optional argument prefix,
// and the working directory it should run in.
//
// Prefix exists because a source checkout is normally driven through uv
// (`uv run github-utils`), and uv resolves the project from its working
// directory. Bin+Prefix together are the command.
type Launcher struct {
	Bin    string
	Prefix []string
	// Dir is the child's working directory; empty inherits the parent's.
	Dir string
}

// Argv prefixes args with the program and its prefix.
func (l Launcher) Argv(args ...string) []string {
	out := make([]string, 0, 1+len(l.Prefix)+len(args))
	out = append(out, l.Bin)
	out = append(out, l.Prefix...)
	return append(out, args...)
}

// Describe renders a command for display.
func (l Launcher) Describe(args ...string) string { return strings.Join(l.Argv(args...), " ") }

// Resolve locates the github-utils CLI.
//
// The search order is: an explicit override, the GITHUB_UTILS_BIN environment
// variable, `github-utils` on PATH, the checkout's own .venv entry point found
// above the working directory or above this executable, and finally
// `uv run github-utils` scoped to a directory whose pyproject.toml declares the
// entry point.
//
// The two starting points matter. This binary is run from wherever a shell
// happens to be, so looking only above the working directory would miss the
// checkout it was built from and fall through to a `uv run` that cannot resolve
// the project either.
func Resolve(override string) (Launcher, error) {
	if explicit := strings.TrimSpace(override); explicit != "" {
		return resolveExplicit(explicit)
	}
	if env := strings.TrimSpace(os.Getenv(EnvBinary)); env != "" {
		return resolveExplicit(env)
	}
	if path, err := exec.LookPath(BinaryName); err == nil {
		return Launcher{Bin: path}, nil
	}

	roots := searchRoots()
	for _, start := range roots {
		if venv := findVenvCLI(start); venv != "" {
			return Launcher{Bin: venv}, nil
		}
	}
	if uv, err := exec.LookPath("uv"); err == nil {
		for _, start := range roots {
			if project := findProjectDir(start); project != "" {
				// uv resolves the project from its working directory, so pin it;
				// otherwise `uv run` cannot find the entry point.
				return Launcher{Bin: uv, Prefix: []string{"run", BinaryName}, Dir: project}, nil
			}
		}
	}
	return Launcher{}, ErrBinaryNotFound
}

// resolveExplicit accepts either a name on PATH or a path to an entry point.
func resolveExplicit(value string) (Launcher, error) {
	if path, err := exec.LookPath(value); err == nil {
		return Launcher{Bin: path}, nil
	}
	if info, err := os.Stat(value); err == nil && !info.IsDir() {
		return Launcher{Bin: value}, nil
	}
	return Launcher{}, fmt.Errorf("%s %q is neither on PATH nor an existing file", EnvBinary, value)
}

// searchRoots is where discovery looks for the checkout: the working directory
// first, then the directory holding this executable, which during development
// is gotui/cmd/github-utils-tui inside the repository.
func searchRoots() []string {
	roots := make([]string, 0, 2)
	if cwd, err := os.Getwd(); err == nil {
		roots = append(roots, cwd)
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		roots = append(roots, filepath.Dir(exe))
	}
	return roots
}

// findVenvCLI walks up from start looking for the checkout's installed entry
// point.
func findVenvCLI(start string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, ".venv", "bin", BinaryName)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// findProjectDir walks up from start looking for the directory that declares the
// CLI console script, which is what `uv run github-utils` needs as its working
// directory.
func findProjectDir(start string) string {
	dir := start
	for {
		candidate := filepath.Join(dir, "pyproject.toml")
		if data, err := os.ReadFile(candidate); err == nil &&
			bytes.Contains(data, []byte(EntryPoint)) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// EnvWith returns the child's environment.
//
// The child inherits the user's own variables (GITHUB_TOKEN, and anything the
// config loader reads) untouched, so it behaves exactly as the Python CLI would.
// Overrides replace rather than append: a duplicate entry would let a stale value
// win by position in some shells.
func EnvWith(overrides map[string]string) []string {
	env := os.Environ()
	if len(overrides) == 0 {
		return env
	}
	out := make([]string, 0, len(env)+len(overrides))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		if _, replaced := overrides[name]; replaced {
			continue
		}
		out = append(out, kv)
	}
	for name, value := range overrides {
		out = append(out, name+"="+value)
	}
	return out
}

// Invocation is a ready-to-exec subprocess call.
type Invocation struct {
	Argv []string
	Env  []string
	Dir  string
}

// CommandLine renders argv for display.
func (i Invocation) CommandLine() string { return strings.Join(i.Argv, " ") }
