package tui

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"

	"github.com/jasperan/github-utils/gotui/internal/cli"
	"github.com/jasperan/github-utils/gotui/internal/grid"
	"github.com/jasperan/github-utils/gotui/internal/run"
)

// Testing approach, and why it looks like this.
//
// These tests hand the model messages directly and only ever execute this
// package's OWN commands. They never drain a huh form's command: huh re-arms the
// text input's cursor blink on every update and each blink tick sleeps for around
// half a second, so a naive drain of a form makes a suite hang for minutes. The
// commands this package builds run a stub runner that returns immediately, so
// executing those is safe and is what proves the form-to-invocation path.

// previewFixture is `paint --pattern wave --preview` output, captured from the CLI.
// The trailing padding Rich adds is omitted; a space and a missing character are
// both intensity 0.
const previewFixture = `
Grid Preview

█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░
▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░
▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░
░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒
 ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓
░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█
▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓
`

// planFixture is the tail of a paint plan, captured from the CLI.
const planFixture = `Total estimated commits: 544
Active days: 136
`

// simulateFixture is the tail of a simulation plan, captured from the CLI.
const simulateFixture = `Total commits: 92
Active days: 26 / 30
Average per active day: 3.5
`

// stubRunner answers by flag so one stub can serve every action.
//
// It is safe for concurrent use and records what it was asked to run, which is how
// the tests assert that nothing was reimplemented: the argv must name the CLI.
type stubRunner struct {
	mu       sync.Mutex
	preview  []byte
	plan     []byte
	simulate []byte
	other    []byte
	err      error
	calls    []cli.Invocation
}

func (s *stubRunner) Run(inv cli.Invocation, _ time.Duration) ([]byte, error) {
	s.mu.Lock()
	s.calls = append(s.calls, inv)
	s.mu.Unlock()

	joined := strings.Join(inv.Argv, " ")
	switch {
	case strings.Contains(joined, "--preview"):
		return []byte(s.preview), s.err
	case strings.Contains(joined, "--dry-run"):
		if strings.Contains(joined, "simulate") {
			return []byte(s.simulate), s.err
		}
		return []byte(s.plan), s.err
	default:
		return []byte(s.other), s.err
	}
}

func (s *stubRunner) recorded() []cli.Invocation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]cli.Invocation(nil), s.calls...)
}

// testOptions builds Options with no checkout and no subprocess.
func testOptions(r run.Runner) Options {
	return Options{
		Launcher: cli.Launcher{Bin: "/checkout/.venv/bin/github-utils"},
		Runner:   r,
		Timeout:  5 * time.Second,
		Version:  "2.0.0",
		Patterns: []string{"checkerboard", "heart", "wave"},
		Profiles: []string{"default", "night-owl", "weekend-warrior"},
	}
}

// press sends a key press and returns the resulting model and command.
func press(t *testing.T, m *Model, key tea.KeyPressMsg) (*Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(key)
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", updated)
	}
	return model, cmd
}

// send delivers any message. It exists for a key RELEASE, which is a different
// type from a press and is exactly what the double-fire guard is about.
func send(t *testing.T, m *Model, msg tea.Msg) (*Model, tea.Cmd) {
	t.Helper()
	updated, cmd := m.Update(msg)
	model, ok := updated.(*Model)
	if !ok {
		t.Fatalf("Update returned %T, want *Model", updated)
	}
	return model, cmd
}

// TestModelStartsOnTheMenu pins the entry state and that the launcher renders.
func TestModelStartsOnTheMenu(t *testing.T) {
	m := New(testOptions(&stubRunner{}))
	_ = m.Init()

	if m.screen != ScreenMenu {
		t.Fatalf("screen = %v, want the menu", m.screen)
	}
	if m.form == nil || m.form.State != huh.StateNormal {
		t.Fatal("the menu form must be built and normal up front")
	}
	if view := m.View().Content; !strings.Contains(view, "What do you want to do?") {
		t.Errorf("the menu should render its choices, got %q", view)
	}
}

// TestSeededFlagOpensTheMatchingScreen keeps a flag from making the user navigate
// a menu first.
func TestSeededFlagOpensTheMatchingScreen(t *testing.T) {
	cases := []struct {
		name string
		seed Seed
		want Screen
	}{
		{"text", Seed{Text: "HELLO"}, ScreenPaintText},
		{"pattern", Seed{Pattern: "wave"}, ScreenPaintPattern},
		{"simulate", Seed{Simulate: true}, ScreenSimulate},
		{"compare", Seed{Before: "a.json", After: "b.json"}, ScreenCompare},
		{"nothing", Seed{}, ScreenMenu},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			opts := testOptions(&stubRunner{})
			opts.Seed = c.seed
			m := New(opts)
			if m.screen != c.want {
				t.Errorf("screen = %v, want %v", m.screen, c.want)
			}
		})
	}
}

// TestSeededValuesSurviveIntoTheAnswers pins that a flag actually reaches the
// field the command is built from.
func TestSeededValuesSurviveIntoTheAnswers(t *testing.T) {
	opts := testOptions(&stubRunner{})
	opts.Seed = Seed{Text: "HELLO", Intensity: "2"}
	m := New(opts)
	_ = m.Init()

	if m.paint.Text != "HELLO" {
		t.Errorf("Text = %q, want the seeded value", m.paint.Text)
	}
	if m.paint.Intensity != "2" {
		t.Errorf("Intensity = %q, want the seeded value", m.paint.Intensity)
	}
}

// TestPaintTextFlowRunsTheCLIAndRendersTheGrid is the end-to-end path: a
// completed paint form must build a CLI invocation, and the result must show the
// parsed grid and the totals.
func TestPaintTextFlowRunsTheCLIAndRendersTheGrid(t *testing.T) {
	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture)}
	opts := testOptions(runner)
	opts.Seed = Seed{Text: "HELLO", Intensity: "4"}
	m := New(opts)
	_ = m.Init()

	// Only this package's own command is executed; it returns immediately.
	msg := m.paintTextCmd()()
	if out, ok := msg.(outputMsg); !ok {
		t.Fatalf("command produced %T, want outputMsg", msg)
	} else if out.err != nil {
		t.Fatalf("paintTextCmd: %v", out.err)
	}

	updated, _ := m.Update(msg)
	m = updated.(*Model)

	if m.stage != stageResult {
		t.Fatalf("stage = %v, want the result stage", m.stage)
	}
	view := m.View().Content
	for _, want := range []string{"Contribution graph", "Total estimated commits", "544", "Active days", "136"} {
		if !strings.Contains(view, want) {
			t.Errorf("result view is missing %q", want)
		}
	}

	// The delegation contract: argv must name the CLI, and it must be the repo's
	// own paint command rather than anything recomputed here.
	calls := runner.recorded()
	if len(calls) != 2 {
		t.Fatalf("ran %d commands, want a preview and a plan", len(calls))
	}
	for _, inv := range calls {
		if !strings.HasPrefix(inv.Argv[0], "/checkout/.venv/bin/github-utils") {
			t.Errorf("argv = %v, want it to run the repo's CLI", inv.Argv)
		}
		if inv.Argv[1] != "paint" {
			t.Errorf("argv = %v, want the paint subcommand", inv.Argv)
		}
	}
}

// TestFailureRendersAnErrorPaneNotAnEmptyGrid is the anti-silent-wrongness rule:
// a broken CLI must look broken, never like an empty calendar.
func TestFailureRendersAnErrorPaneNotAnEmptyGrid(t *testing.T) {
	runner := &stubRunner{err: errors.New("Error: Pattern 'nope' not found as file or builtin pattern")}
	opts := testOptions(runner)
	opts.Seed = Seed{Pattern: "nope"}
	m := New(opts)
	_ = m.Init()

	msg := m.paintPatternCmd()()
	updated, _ := m.Update(msg)
	m = updated.(*Model)

	view := m.View().Content
	if !strings.Contains(view, "The CLI reported a failure") {
		t.Errorf("view should report a failure, got %q", view)
	}
	if strings.Contains(view, "Contribution graph") {
		t.Error("view rendered a grid panel for a failed run")
	}
}

// TestUnparseablePreviewRendersAnErrorRatherThanAnEmptyGrid is the same rule for a
// preview whose glyphs changed: an empty calendar would look like a real answer.
func TestUnparseablePreviewRendersAnErrorRatherThanAnEmptyGrid(t *testing.T) {
	runner := &stubRunner{
		preview: []byte("Grid Preview\n\n" + strings.Repeat("ZZZ\n", grid.Rows)),
		plan:    []byte(planFixture),
	}
	opts := testOptions(runner)
	opts.Seed = Seed{Text: "HELLO"}
	m := New(opts)
	_ = m.Init()

	msg := m.paintTextCmd()()
	updated, _ := m.Update(msg)
	m = updated.(*Model)

	if view := m.View().Content; !strings.Contains(view, "Grid preview") {
		t.Errorf("view should report the grid parse failure, got %q", view)
	}
}

// TestSimulateFlowRendersEveryTotal pins the three simulate totals, including the
// two-number active-days line.
func TestSimulateFlowRendersEveryTotal(t *testing.T) {
	runner := &stubRunner{simulate: []byte(simulateFixture)}
	opts := testOptions(runner)
	opts.Seed = Seed{Simulate: true, Days: "30", Profile: "night-owl", Seed: "7"}
	m := New(opts)
	_ = m.Init()

	msg := m.simulateCmd()()
	updated, _ := m.Update(msg)
	m = updated.(*Model)

	view := m.View().Content
	for _, want := range []string{"Total commits", "92", "Active days", "26 / 30", "Average per active day", "3.5"} {
		if !strings.Contains(view, want) {
			t.Errorf("result view is missing %q", want)
		}
	}
	if calls := runner.recorded(); len(calls) != 1 || calls[0].Argv[1] != "simulate" {
		t.Errorf("calls = %v, want a single simulate run", calls)
	}
}

// TestValidationFailuresNeverReachTheCLI keeps a bad form value from being sent.
func TestValidationFailuresNeverReachTheCLI(t *testing.T) {
	cases := []struct {
		name  string
		build func(m *Model, opts *Options)
	}{
		{"empty text", func(m *Model, _ *Options) { m.paint.Text = "   " }},
		{"text too long", func(m *Model, _ *Options) { m.paint.Text = "ABCDEFGHIJ" }},
		{"days below one", func(m *Model, _ *Options) { m.sim.Days = "0" }},
		{"seed not a number", func(m *Model, _ *Options) { m.sim.Days = "30"; m.sim.Seed = "abc" }},
		{"missing snapshot", func(m *Model, _ *Options) { m.cmp.Before = "nope.json"; m.cmp.After = "nope2.json" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			runner := &stubRunner{}
			opts := testOptions(runner)
			m := New(opts)
			c.build(m, &opts)

			// Every builder reports a validation failure as a message, so it renders in
			// the same panel as a failed subprocess. Executing them is safe: each one
			// returns immediately (there is no form command to drain).
			for _, cmd := range []tea.Cmd{m.paintTextCmd(), m.simulateCmd(), m.compareCmd()} {
				if cmd == nil {
					t.Error("a builder returned no command")
					continue
				}
				msg := cmd()
				if out, ok := msg.(outputMsg); !ok || out.err == nil {
					t.Errorf("msg = %#v, want a validation failure", msg)
				}
			}
			if len(runner.recorded()) != 0 {
				t.Errorf("the CLI was invoked despite a validation failure: %v", runner.recorded())
			}
		})
	}
}

// TestOnlyKeyPressesAreActedOn is the regression guard for this workspace's known
// bug: bubbletea v2 delivers key releases as well as presses, and acting on both
// fires every binding twice.
func TestOnlyKeyPressesAreActedOn(t *testing.T) {
	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture)}
	opts := testOptions(runner)
	opts.Seed = Seed{Text: "HELLO"}
	m := New(opts)
	_ = m.Init()

	// Put the model on the result stage so a key has something to move.
	updated, _ := m.Update(m.paintTextCmd()())
	m = updated.(*Model)
	m.viewport.GotoBottom()
	before := m.viewport.YOffset()

	// A release for a scrolling key must be ignored entirely.
	after, _ := send(t, m, tea.KeyReleaseMsg{Text: "k", Code: 'k'})
	if after.viewport.YOffset() != before {
		t.Errorf("a key RELEASE scrolled the viewport %d -> %d; bindings would fire twice",
			before, after.viewport.YOffset())
	}
}

// TestEveryWidthRendersWithoutPanicking sweeps the whole range the task calls out,
// including 0 and below, where a negative content width used to panic a renderer.
func TestEveryWidthRendersWithoutPanicking(t *testing.T) {
	runner := &stubRunner{preview: []byte(previewFixture), plan: []byte(planFixture), simulate: []byte(simulateFixture)}

	for _, stage := range []string{"form", "running", "result"} {
		for width := -5; width <= 120; width++ {
			opts := testOptions(runner)
			opts.Seed = Seed{Text: "HELLO"}
			m := New(opts)
			if err := render(m, stage, width); err != nil {
				t.Fatalf("width %d at stage %s: render panicked: %v", width, stage, err)
			}
		}
	}
}

// render drives one model to a stage at a width, recovering so a panic is a tidy
// test failure rather than a crashed binary.
func render(m *Model, stage string, width int) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(panicMessage(r))
		}
	}()
	m.Update(tea.WindowSizeMsg{Width: width, Height: 20})
	switch stage {
	case "running":
		m.stage = stageRunning
		m.body = "Running the github-utils CLI…"
	case "result":
		updated, _ := m.Update(m.paintTextCmd()())
		mm := updated.(*Model)
		m.stage = mm.stage
		m.body = mm.body
		m.viewport = mm.viewport
		m.title = mm.title
		m.commands = mm.commands
	}
	_ = m.View()
	return nil
}

// panicMessage renders a recovered panic value.
func panicMessage(r any) string {
	if e, ok := r.(error); ok {
		return e.Error()
	}
	if s, ok := r.(string); ok {
		return s
	}
	return "panic"
}

// TestAboutIsLocal pins that the About panel runs nothing: describing the front-end
// must not cost a subprocess.
func TestAboutIsLocal(t *testing.T) {
	runner := &stubRunner{}
	opts := testOptions(runner)
	m := New(opts)
	m.menu.Choice = ActionAbout

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if cmd != nil {
		if msg := cmd(); msg != nil {
			// A huh form's completion command is expected here and is not drained;
			// what matters is that no CLI call was made.
			_ = msg
		}
	}
	mm := updated.(*Model)
	_ = mm.advance()

	if len(runner.recorded()) != 0 {
		t.Errorf("About ran %d commands; it must be purely local", len(runner.recorded()))
	}
}

// TestSelectSeedNeverSilentlyClobbers is the guard for the huh Select trap.
//
// The mechanism, measured: huh.NewSelect().Value(&v) writes the selection back
// IMMEDIATELY at construction, so a bound value that is not among the options is
// replaced by the first option before the form is ever shown. Clearing the value
// beforehand therefore cannot help; the rejected request has to be recorded and
// refused, which is what Unavailable is for.
func TestSelectSeedNeverSilentlyClobbers(t *testing.T) {
	patterns := []string{"checkerboard", "heart", "wave"}

	t.Run("an unknown pattern is recorded, not substituted", func(t *testing.T) {
		ans := PaintPatternAnswers{Pattern: "does-not-exist"}
		PaintPatternForm(&ans, patterns)

		if ans.Unavailable != "does-not-exist" {
			t.Errorf("Unavailable = %q, want the rejected request recorded", ans.Unavailable)
		}
		// huh will have substituted the first option into the bound field; that is
		// expected and harmless ONLY because the rejection above is checked first.
		runner := &stubRunner{}
		m := New(testOptions(runner))
		m.ptrn = ans
		assertRefused(t, m.paintPatternCmd())
		if calls := runner.recorded(); len(calls) != 0 {
			t.Errorf("the CLI was invoked %v for an unavailable pattern, want no call", calls)
		}
	})

	t.Run("a known pattern is kept", func(t *testing.T) {
		ans := PaintPatternAnswers{Pattern: "wave"}
		PaintPatternForm(&ans, patterns)
		if ans.Pattern != "wave" {
			t.Errorf("Pattern = %q, want the requested pattern", ans.Pattern)
		}
		if ans.Unavailable != "" {
			t.Errorf("Unavailable = %q, want empty for a valid pattern", ans.Unavailable)
		}
	})

	t.Run("an empty pattern defaults to a real option", func(t *testing.T) {
		ans := PaintPatternAnswers{}
		PaintPatternForm(&ans, patterns)
		if ans.Pattern != "checkerboard" {
			t.Errorf("Pattern = %q, want the first real option", ans.Pattern)
		}
		if ans.Unavailable != "" {
			t.Errorf("Unavailable = %q, want empty when nothing was requested", ans.Unavailable)
		}
	})

	t.Run("an unknown profile is recorded, not substituted", func(t *testing.T) {
		ans := SimulateAnswers{Profile: "nope", Days: "30"}
		SimulateForm(&ans, []string{"default", "night-owl"}, "default")
		if ans.Unavailable != "nope" {
			t.Errorf("Unavailable = %q, want the rejected request recorded", ans.Unavailable)
		}
		runner := &stubRunner{}
		m := New(testOptions(runner))
		m.sim = ans
		assertRefused(t, m.simulateCmd())
		if calls := runner.recorded(); len(calls) != 0 {
			t.Errorf("the CLI was invoked %v for an unavailable profile, want no call", calls)
		}
	})

	t.Run("an unknown intensity falls back to the documented default", func(t *testing.T) {
		ans := PaintTextAnswers{Intensity: "9"}
		PaintTextForm(&ans)
		if ans.Intensity != DefaultIntensityValue {
			t.Errorf("Intensity = %q, want %q", ans.Intensity, DefaultIntensityValue)
		}
	})
}

// assertRefused runs a command and requires it to report a failure rather than a
// successful result.
func assertRefused(t *testing.T, cmd tea.Cmd) {
	t.Helper()
	if cmd == nil {
		t.Fatal("no command returned")
	}
	msg, ok := cmd().(outputMsg)
	if !ok {
		t.Fatalf("msg = %T, want outputMsg", cmd())
	}
	if msg.err == nil {
		t.Error("expected a refusal, got a successful result")
	}
}

// TestSelectableValueDirectly pins the helper's contract.
//
// An unavailable request still yields the first option, because huh will write that
// back regardless; what matters is that rejectedValue reports the request so the
// command builder can refuse it.
func TestSelectableValueDirectly(t *testing.T) {
	options := []huh.Option[string]{
		huh.NewOption("a", "a"),
		huh.NewOption("b", "b"),
	}
	if got := selectableValue("b", options); got != "b" {
		t.Errorf("selectableValue = %q, want b (available)", got)
	}
	if got := selectableValue("", options); got != "a" {
		t.Errorf("selectableValue = %q, want the first option when nothing was asked for", got)
	}
	if got := selectableValue("zzz", options); got != "a" {
		t.Errorf("selectableValue = %q, want the first option, since huh writes it back anyway", got)
	}
	if got := selectableValue("a", nil); got != "" {
		t.Errorf("selectableValue = %q, want empty for no options", got)
	}
}

// TestRejectedValueDirectly pins the detector that makes the refusal possible.
func TestRejectedValueDirectly(t *testing.T) {
	options := []huh.Option[string]{
		huh.NewOption("a", "a"),
		huh.NewOption("b", "b"),
	}
	if got := rejectedValue("b", options); got != "" {
		t.Errorf("rejectedValue(b) = %q, want empty for an available value", got)
	}
	if got := rejectedValue("", options); got != "" {
		t.Errorf("rejectedValue(empty) = %q, want empty: no preference is not a rejection", got)
	}
	if got := rejectedValue("   ", options); got != "" {
		t.Errorf("rejectedValue(blank) = %q, want empty", got)
	}
	if got := rejectedValue("zzz", options); got != "zzz" {
		t.Errorf("rejectedValue(zzz) = %q, want the rejected request", got)
	}
}

// TestRunAgainReopensTheRightForm pins that the tracking field is used rather than
// an inferred one, and that an About panel does not send the user to a paint form.
func TestRunAgainReopensTheRightForm(t *testing.T) {
	runner := &stubRunner{simulate: []byte(simulateFixture)}
	opts := testOptions(runner)
	opts.Seed = Seed{Simulate: true}
	m := New(opts)
	_ = m.Init()

	updated, _ := m.Update(m.simulateCmd()())
	m = updated.(*Model)
	if m.resultScreen != ScreenSimulate {
		t.Fatalf("resultScreen = %v, want the simulate screen", m.resultScreen)
	}

	m.runAgain()
	if m.screen != ScreenSimulate || m.stage != stageForm {
		t.Errorf("runAgain left screen=%v stage=%v, want the simulate form", m.screen, m.stage)
	}
}

// TestEscapeFromTheMenuBackToTheMenu keeps Esc from quitting the program on the
// menu, where there is nowhere to go back to.
func TestEscapeFromTheMenuBackToTheMenu(t *testing.T) {
	m := New(testOptions(&stubRunner{}))
	_ = m.Init()

	updated, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	mm := updated.(*Model)
	if mm.stage != stageForm {
		t.Errorf("stage = %v, want the form stage", mm.stage)
	}
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("esc on the menu quit the program")
		}
	}
}

// TestEscClosesAnOpenForm is the regression guard for a bug that has shipped
// twice in this workspace.
//
// huh's default keymap binds Quit to ctrl+c ONLY, and the sole path that sets
// StateAborted is key.Matches(msg, keymap.Quit). Escape therefore never aborts a
// huh form, so delegating esc to the form left it installed: it kept the focus
// and silently swallowed every later keystroke, while the footer advertised
// "esc back". The last assertion is the one that catches that damage.
func TestEscClosesAnOpenForm(t *testing.T) {
	m := New(testOptions(&stubRunner{}))
	_ = m.Init()
	m = openPaintTextForm(t, m)

	if m.form == nil {
		t.Fatal("the paint-text form is not open; the test cannot prove anything")
	}
	if m.notice != "" {
		t.Fatalf("notice = %q before esc, want empty", m.notice)
	}
	if view := m.View().Content; !strings.Contains(view, "Text to paint") {
		t.Fatalf("the paint-text form is not what is on screen: %q", view)
	}
	paintForm := m.form

	m, _ = press(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})

	// This repo's launcher is itself a huh form, so "closed" cannot mean
	// form == nil. It means the paint form was replaced by the menu's, which is what
	// toMenu rebuilds. Identity and the rendered screen both prove that.
	if m.form == paintForm {
		t.Fatal("esc left the paint form installed; it would keep swallowing keystrokes")
	}
	if m.form == nil || m.form.State != huh.StateNormal {
		t.Fatalf("no live form after esc: formNil=%v", m.form == nil)
	}
	if m.notice == "" || !strings.Contains(m.notice, "Cancelled") {
		t.Errorf("notice = %q, want a cancellation notice", m.notice)
	}
	// "esc back": the paint form is left behind, not merely hidden.
	if m.screen != ScreenMenu {
		t.Errorf("screen = %v, want the menu after esc", m.screen)
	}
	if m.stage != stageForm {
		t.Errorf("stage = %v, want the form stage", m.stage)
	}
	view := m.View().Content
	if strings.Contains(view, "Text to paint") {
		t.Errorf("the paint form is still on screen after esc: %q", view)
	}
	if !strings.Contains(view, "What do you want to do?") {
		t.Errorf("the menu is not on screen after esc: %q", view)
	}

	// The damage this guards against: if the dead form were still installed and
	// holding focus, this navigation would be swallowed.
	m = openPaintTextForm(t, m)
	if m.screen != ScreenPaintText || m.form == nil {
		t.Fatalf("navigation after esc was swallowed: screen=%v formNil=%v", m.screen, m.form == nil)
	}
	if got := m.View().Content; !strings.Contains(got, "Text to paint") {
		t.Errorf("the paint form did not reopen after esc: %q", got)
	}
}

// TestEscOnTheMenuIsNotTheOnlyWayOut pins that esc with no form open still does
// not quit the program, so the esc intercept cannot regress the menu.
func TestEscWithNoFormDoesNotQuit(t *testing.T) {
	m := New(testOptions(&stubRunner{}))
	_ = m.Init()
	m.form = nil

	m, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd != nil {
		if _, quit := cmd().(tea.QuitMsg); quit {
			t.Error("esc with no form open quit the program")
		}
	}
	if m.form != nil {
		t.Error("esc with no form open opened one")
	}
}

// openPaintTextForm drives the launcher the way a user does: submit the menu's
// default choice ("Paint text on the graph") and deliver the messages huh needs
// to finish that group.
//
// Only the messages huh itself produces for the menu are delivered, and it stops
// the moment the new screen is up. The new form's own command is deliberately NOT
// run: huh re-arms a cursor blink on every update and each blink tick sleeps for
// about half a second, so draining a form's commands hangs this suite (see the
// testing note at the top of this file).
func openPaintTextForm(t *testing.T, m *Model) *Model {
	t.Helper()
	m, cmd := press(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	for i := 0; i < 4 && cmd != nil && m.screen != ScreenPaintText; i++ {
		msg := cmd()
		if msg == nil {
			break
		}
		m, cmd = send(t, m, msg)
	}
	if m.screen != ScreenPaintText {
		t.Fatalf("the menu did not open the paint-text form: screen=%v", m.screen)
	}
	if m.form == nil {
		t.Fatal("the paint-text form was not built")
	}
	return m
}

// TestFooterHintNeverWraps keeps a narrow terminal from pushing the hint onto a
// second line and shifting the frame.
//
// The measurement is lipgloss.Width, not a rune count: the rendered hint carries
// ANSI colour sequences, and counting those as width would fail on every styled
// string while the real terminal sees something far narrower.
func TestFooterHintNeverWraps(t *testing.T) {
	hint := "r run again · esc back to the menu · q quit · up/down scroll"
	for _, width := range []int{0, 1, 5, 20, 200} {
		got := Footer(hint, width)
		if width >= 2 && lipgloss.Width(got) > width {
			t.Errorf("Footer(width=%d) is %d columns, want at most %d", width, lipgloss.Width(got), width)
		}
	}
}

// TestHeatmapReportsTruncation keeps a narrow terminal from silently hiding weeks.
func TestHeatmapReportsTruncation(t *testing.T) {
	g, err := grid.ParsePreview([]byte(previewFixture))
	if err != nil {
		t.Fatalf("ParsePreview: %v", err)
	}

	full, columns, truncated := Heatmap(g, grid.Cols)
	if columns != grid.Cols || truncated != 0 {
		t.Errorf("columns=%d truncated=%d, want the full width", columns, truncated)
	}
	if lines := strings.Split(full, "\n"); len(lines) != grid.Rows {
		t.Errorf("rendered %d lines, want %d", len(lines), grid.Rows)
	}

	narrow, columns, truncated := Heatmap(g, 20)
	if columns != 20 || truncated != grid.Cols-20 {
		t.Errorf("columns=%d truncated=%d, want 20 and %d", columns, truncated, grid.Cols-20)
	}
	if lipgloss.Width(strings.Split(narrow, "\n")[0]) != 20 {
		t.Errorf("narrow row is %d columns, want 20", lipgloss.Width(strings.Split(narrow, "\n")[0]))
	}
}

// TestPanesRespectTheChrome pins the measured border+padding width, since getting
// it wrong makes every box wrap at the terminal edge.
func TestPanesRespectTheChrome(t *testing.T) {
	for _, total := range []int{10, 26, 40, 80} {
		pane := Pane("Title", "body", total)
		widest := 0
		for _, line := range strings.Split(pane, "\n") {
			if w := lipgloss.Width(line); w > widest {
				widest = w
			}
		}
		want := max(total, minPaneTotal)
		if widest != want {
			t.Errorf("Pane(total=%d) renders %d columns, want %d", total, widest, want)
		}
	}
}
