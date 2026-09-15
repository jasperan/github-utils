package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeCLI creates a checkout-shaped directory holding an executable entry
// point, which is what discovery looks for.
func writeFakeCLI(t *testing.T, root string) string {
	t.Helper()
	dir := filepath.Join(root, ".venv", "bin")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(dir, BinaryName)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// TestResolveFindsTheCheckoutVenv pins the primary discovery route: walking up
// from the working directory to a checkout's own entry point.
func TestResolveFindsTheCheckoutVenv(t *testing.T) {
	root := t.TempDir()
	want := writeFakeCLI(t, root)
	nested := filepath.Join(root, "gotui", "cmd")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	t.Chdir(nested)

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Bin != want {
		t.Errorf("Bin = %q, want %q", got.Bin, want)
	}
}

// TestResolveHonoursExplicitPath pins the documented escape hatch.
func TestResolveHonoursExplicitPath(t *testing.T) {
	root := t.TempDir()
	path := writeFakeCLI(t, root)

	got, err := Resolve(path)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Bin != path {
		t.Errorf("Bin = %q, want %q", got.Bin, path)
	}
}

// TestResolveHonoursEnvOverride pins GITHUB_UTILS_BIN.
func TestResolveHonoursEnvOverride(t *testing.T) {
	root := t.TempDir()
	path := writeFakeCLI(t, root)
	t.Setenv(EnvBinary, path)

	got, err := Resolve("")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got.Bin != path {
		t.Errorf("Bin = %q, want %q", got.Bin, path)
	}
}

// TestResolveRejectsAMissingOverride checks the error names the variable the user
// just set, because that is the thing they can act on.
func TestResolveRejectsAMissingOverride(t *testing.T) {
	t.Setenv(EnvBinary, filepath.Join(t.TempDir(), "does-not-exist"))

	_, err := Resolve("")
	if err == nil {
		t.Fatal("Resolve returned no error for a nonexistent override")
	}
	if !strings.Contains(err.Error(), EnvBinary) {
		t.Errorf("error = %q, want it to name %s", err, EnvBinary)
	}
}

// TestLauncherArgvPutsThePrefixFirst pins the uv-shaped command, where the prefix
// must come before the subcommand.
func TestLauncherArgvPutsThePrefixFirst(t *testing.T) {
	l := Launcher{Bin: "uv", Prefix: []string{"run", BinaryName}, Dir: "/checkout"}

	got := l.Argv("paint", "HELLO")
	want := []string{"uv", "run", BinaryName, "paint", "HELLO"}
	if len(got) != len(want) {
		t.Fatalf("Argv = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Argv = %v, want %v", got, want)
		}
	}
	if described := l.Describe("paint"); !strings.Contains(described, "run "+BinaryName) {
		t.Errorf("Describe = %q, want it to show the prefix", described)
	}
}

// TestTextPreviewPinsArgvAndWidth pins the exact argv plus the pinned COLUMNS.
//
// The width matters: Rich falls back to 80 columns on a pipe and honours COLUMNS
// otherwise, and a narrow inherited value would wrap each 52-column grid row onto
// two lines, silently breaking the parse.
func TestTextPreviewPinsArgvAndWidth(t *testing.T) {
	t.Setenv("COLUMNS", "40")
	inv := TextPreview(Launcher{Bin: "github-utils"}, "HELLO", 3)

	want := []string{"github-utils", "paint", "HELLO", "--preview", "--intensity", "3"}
	if strings.Join(inv.Argv, " ") != strings.Join(want, " ") {
		t.Errorf("Argv = %v, want %v", inv.Argv, want)
	}
	if got := envValue(inv.Env, "COLUMNS"); got != previewColumns {
		t.Errorf("COLUMNS = %q, want %q (a narrow value would wrap the grid)", got, previewColumns)
	}
}

// TestTextPreviewOmitsAZeroIntensity keeps the CLI's own default in play when the
// caller has no opinion, rather than pinning 0, which paint rejects.
func TestTextPreviewOmitsAZeroIntensity(t *testing.T) {
	inv := TextPreview(Launcher{Bin: "github-utils"}, "HELLO", 0)
	for _, arg := range inv.Argv {
		if arg == "--intensity" {
			t.Fatalf("Argv = %v, want no --intensity for an unset value", inv.Argv)
		}
	}
}

// TestOneInputSourcePerPaintCall pins the CLI's own rule: paint takes exactly one
// of a text argument, --pattern or --file, and rejects a combination.
func TestOneInputSourcePerPaintCall(t *testing.T) {
	cases := []struct {
		name string
		inv  Invocation
	}{
		{"text", TextPreview(Launcher{Bin: "g"}, "HI", 4)},
		{"pattern", PatternPreview(Launcher{Bin: "g"}, "wave")},
		{"file", FilePreview(Launcher{Bin: "g"}, "p.yaml")},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hasPattern, hasFile := false, false
			for _, arg := range c.inv.Argv {
				switch arg {
				case "--pattern":
					hasPattern = true
				case "--file":
					hasFile = true
				}
			}
			if hasPattern && hasFile {
				t.Errorf("Argv = %v, want at most one input source", c.inv.Argv)
			}
			if strings.Join(c.inv.Argv, " ") == "" {
				t.Error("argv is empty")
			}
		})
	}
}

// TestPatternCatalogPinsTheBarePaintCall pins how the pattern list is obtained.
//
// There is no listing subcommand: a bare `paint` prints the catalogue and exits 1,
// so the argv must stay bare and the runner must read stdout on a non-zero exit.
func TestPatternCatalogPinsTheBarePaintCall(t *testing.T) {
	inv := PatternCatalog(Launcher{Bin: "github-utils"})
	if len(inv.Argv) != 2 || inv.Argv[0] != "github-utils" || inv.Argv[1] != "paint" {
		t.Errorf("Argv = %v, want a bare `github-utils paint`", inv.Argv)
	}
}

// TestSimulatePlanRejectsAnEmptyWindow keeps the CLI's own minimum (>=1 day).
func TestSimulatePlanRejectsAnEmptyWindow(t *testing.T) {
	if _, err := SimulatePlan(Launcher{Bin: "g"}, SimulateOptions{Days: 0}); err == nil {
		t.Fatal("SimulatePlan accepted zero days; the CLI requires at least 1")
	}
}

// TestSimulatePlanSeedPointerSemantics is the bug this pins: a nil seed must omit
// the flag (random plan) while a ZERO seed must be passed through, because 0 is a
// legitimate seed and omitting it would silently produce a different plan.
func TestSimulatePlanSeedPointerSemantics(t *testing.T) {
	random, err := SimulatePlan(Launcher{Bin: "g"}, SimulateOptions{Days: 30, DryRun: true})
	if err != nil {
		t.Fatalf("SimulatePlan: %v", err)
	}
	if strings.Contains(strings.Join(random.Argv, " "), "--seed") {
		t.Errorf("Argv = %v, want no --seed for a nil seed", random.Argv)
	}

	zero := 0
	pinned, err := SimulatePlan(Launcher{Bin: "g"}, SimulateOptions{Days: 30, Seed: &zero, DryRun: true})
	if err != nil {
		t.Fatalf("SimulatePlan: %v", err)
	}
	if !strings.Contains(strings.Join(pinned.Argv, " "), "--seed 0") {
		t.Errorf("Argv = %v, want --seed 0 passed through", pinned.Argv)
	}
}

// TestSimulatePlanOmitsAnEmptyProfile keeps the CLI's default profile in play.
func TestSimulatePlanOmitsAnEmptyProfile(t *testing.T) {
	inv, err := SimulatePlan(Launcher{Bin: "g"}, SimulateOptions{Days: 30, Profile: "  "})
	if err != nil {
		t.Fatalf("SimulatePlan: %v", err)
	}
	if strings.Contains(strings.Join(inv.Argv, " "), "--profile") {
		t.Errorf("Argv = %v, want no --profile for a blank value", inv.Argv)
	}
}

// TestCompareRequiresBothFiles pins that a half-specified comparison is refused
// here rather than by the CLI after a round trip.
func TestCompareRequiresBothFiles(t *testing.T) {
	if _, err := Compare(Launcher{Bin: "g"}, CompareOptions{Before: "a.json"}); err == nil {
		t.Fatal("Compare accepted a missing --after")
	}
	if _, err := Compare(Launcher{Bin: "g"}, CompareOptions{After: "b.json"}); err == nil {
		t.Fatal("Compare accepted a missing --before")
	}
	inv, err := Compare(Launcher{Bin: "g"}, CompareOptions{Before: "a.json", After: "b.json"})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	want := "g stats compare --before a.json --after b.json"
	if got := inv.CommandLine(); got != want {
		t.Errorf("CommandLine = %q, want %q", got, want)
	}
}

// TestEnvWithReplacesRatherThanAppends pins the environment contract: a duplicate
// entry would let a stale value win by position in some shells.
func TestEnvWithReplacesRatherThanAppends(t *testing.T) {
	t.Setenv("COLUMNS", "20")
	env := EnvWith(map[string]string{"COLUMNS": "200"})

	count := 0
	for _, kv := range env {
		if strings.HasPrefix(kv, "COLUMNS=") {
			count++
			if kv != "COLUMNS=200" {
				t.Errorf("found %q, want COLUMNS=200", kv)
			}
		}
	}
	if count != 1 {
		t.Errorf("COLUMNS appears %d times, want exactly 1", count)
	}
}

// TestEnvWithInheritsUnrelatedVariables keeps the child's own config and token
// variables in play, so it behaves as the Python CLI would.
func TestEnvWithInheritsUnrelatedVariables(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "secret-value")
	env := EnvWith(nil)
	if envValue(env, "GITHUB_TOKEN") != "secret-value" {
		t.Error("EnvWith dropped an inherited variable the CLI reads")
	}
}

// envValue reads a variable out of an exec-style environment.
func envValue(env []string, name string) string {
	for _, kv := range env {
		if key, value, ok := strings.Cut(kv, "="); ok && key == name {
			return value
		}
	}
	return ""
}
