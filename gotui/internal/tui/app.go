// Package tui is the Go front-end's terminal UI, built on charm.land/bubbletea/v2
// and charm.land/huh/v2.
//
// It paints nothing and computes nothing. Every glyph, pattern, profile and
// commit estimate on screen was produced by the repository's own `github-utils`
// CLI, which this front-end runs and renders, so a Go user and a Python user get
// identical results.
package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/jasperan/github-utils/gotui/internal/cli"
	"github.com/jasperan/github-utils/gotui/internal/grid"
	"github.com/jasperan/github-utils/gotui/internal/parse"
	"github.com/jasperan/github-utils/gotui/internal/run"
)

// Screen identifies the active form.
type Screen int

// Screens.
const (
	ScreenMenu Screen = iota
	ScreenPaintText
	ScreenPaintPattern
	ScreenSimulate
	ScreenCompare
	ScreenAbout
)

// DefaultTimeout bounds one delegated CLI call.
const DefaultTimeout = run.DefaultTimeout

// stage separates the three kinds of frame the UI shows.
//
// A result is NOT a screen: it is a stage the user can only scroll, run again, or
// leave. Keeping a continue form out of the result frame is what avoids a key
// conflict between the viewport's scrolling keys and a form's own arrow handling.
type stage int

const (
	stageForm stage = iota
	stageRunning
	stageResult
)

// Seed carries command-line flags so a whole action can run without a prompt.
type Seed struct {
	Text      string
	Pattern   string
	Intensity string
	Simulate  bool
	Days      string
	Profile   string
	Seed      string
	Before    string
	After     string
}

// Options configure a TUI run.
type Options struct {
	// Launcher is the resolved CLI. Its command line is shown in the UI so the
	// user can always see what will actually run.
	Launcher cli.Launcher
	// Runner executes invocations; injectable so tests need no checkout.
	Runner run.Runner
	// Timeout bounds each call.
	Timeout time.Duration
	// Version is the CLI's reported version, shown in the header and the menu.
	Version string
	// Patterns and Profiles are catalogues read from the CLI at startup.
	Patterns []string
	Profiles []string
	// DefaultProfile is the profile `simulate` uses when it is not told one, read
	// from the CLI's own help so the form starts where the Python CLI would.
	DefaultProfile string
	// Seed pre-answers the forms from flags.
	Seed Seed
}

// errNoRunner is reported in the result panel when no runner was supplied.
var errNoRunner = errors.New("no runner configured")

// outputMsg carries a finished subprocess back into the update loop.
//
// The raw bytes travel, not a rendered string: rendering happens where the
// current width is known, so a resize reflows an already-loaded result instead of
// leaving it stuck at the old width.
type outputMsg struct {
	// kind is the action that produced this output.
	kind string
	// raw is the text shown verbatim (the CLI's own table or message).
	raw []byte
	// preview is the `--preview` output, when the action produced a grid.
	preview []byte
	// commands are the command lines run, for display.
	commands []string
	err      error
}

// Model is the root bubbletea model.
//
// It is a pointer model on purpose: every huh field binds through a pointer, and
// a value-typed Bubble Tea model would silently persist defaults because Update
// receives a copy.
type Model struct {
	width  int
	height int
	opts   Options

	stage  stage
	screen Screen
	form   *huh.Form

	// resultScreen is the screen that produced the current result, so "run again"
	// reopens the right form. It is tracked explicitly rather than inferred: an
	// About panel is produced by the menu, and inferring from the output kind
	// would send "run again" to a paint form the user never opened.
	resultScreen Screen

	menu  MenuAnswers
	paint PaintTextAnswers
	ptrn  PaintPatternAnswers
	sim   SimulateAnswers
	cmp   CompareAnswers

	viewport viewport.Model

	title    string
	body     string
	commands []string
	failure  error
	notice   string

	// last is the most recent finished output, kept so a resize can re-render it.
	last *outputMsg
}

// New builds the root model and shows the screen the flags asked for.
func New(opts Options) *Model {
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if len(opts.Patterns) == 0 {
		opts.Patterns = defaultPatterns()
	}
	if len(opts.Profiles) == 0 {
		opts.Profiles = defaultProfiles()
	}
	m := Model{
		width:    100,
		height:   30,
		opts:     opts,
		stage:    stageForm,
		screen:   ScreenMenu,
		viewport: viewport.New(viewport.WithWidth(96), viewport.WithHeight(20)),
	}
	m.screen = seededScreen(opts.Seed)
	m.form = m.buildForm()
	m.resize()
	return &m
}

// defaultPatterns and defaultProfiles are the fallbacks used only when the
// startup probe could not read the live catalogues, so the forms still render.
// They are deliberately the values the CLI ships today; a probe that succeeded
// always wins, so a change on the Python side is picked up automatically.
func defaultPatterns() []string { return []string{"checkerboard", "heart", "wave"} }

func defaultProfiles() []string {
	return []string{"default", "night-owl", "weekend-warrior", "9-to-5", "open-source-maintainer"}
}

// seededScreen chooses the opening screen from the flags.
//
// A flag that clearly names one action opens that action's form directly, so
// `github-utils-tui --paint HELLO` does not make the user navigate a menu first.
func seededScreen(s Seed) Screen {
	switch {
	case strings.TrimSpace(s.Pattern) != "":
		return ScreenPaintPattern
	case strings.TrimSpace(s.Text) != "":
		return ScreenPaintText
	case s.Simulate:
		return ScreenSimulate
	case strings.TrimSpace(s.Before) != "" || strings.TrimSpace(s.After) != "":
		return ScreenCompare
	default:
		return ScreenMenu
	}
}

// buildForm builds the form for the active screen.
//
// Every screen transition goes through here, so no form can be left at huh's
// default zero width (which renders as blank lines).
func (m *Model) buildForm() *huh.Form {
	switch m.screen {
	case ScreenPaintText:
		return PaintTextForm(&m.paint)
	case ScreenPaintPattern:
		return PaintPatternForm(&m.ptrn, m.opts.Patterns)
	case ScreenSimulate:
		return SimulateForm(&m.sim, m.opts.Profiles, m.opts.DefaultProfile)
	case ScreenCompare:
		return CompareForm(&m.cmp)
	default:
		return MenuForm(&m.menu, m.opts.Version)
	}
}

// setForm installs a freshly built form, sizes it, and returns its first command.
func (m *Model) setForm(form *huh.Form) tea.Cmd {
	m.form = form
	m.resize()
	return m.form.Init()
}

// resize applies the current window size to the form and the viewport.
//
// huh.NewForm leaves the form at width 0 until something sets it, and a zero-width
// field renders as blank lines. Both dimensions are clamped, so a terminal
// narrower or shorter than the minimum cannot produce a negative size.
func (m *Model) resize() {
	if m.form != nil {
		width := max(m.width-4, minPaneWidth)
		height := max(m.height-2, 5)
		m.form = m.form.WithWidth(width).WithHeight(height)
	}
	m.viewport.SetWidth(max(m.width-2, minPaneWidth))
	m.viewport.SetHeight(m.viewportHeight())
}

// viewportHeight is the rows left for the CLI's own output.
//
// It is computed rather than fixed because the summary block above it grows and
// shrinks with the action: reserving a constant height would either clip the plan
// table or push the footer off the screen at a small terminal size.
func (m *Model) viewportHeight() int {
	used := 0
	if m.stage == stageResult {
		// Header, title, and the command hint each occupy a line plus a separator.
		used = 6
		if m.title != "" {
			used += 2
		}
		if len(m.commands) > 0 {
			used += len(m.commands) + 1
		}
	}
	if m.body != "" {
		used += strings.Count(m.body, "\n") + 2
	}
	height := m.height - used
	if height < 3 {
		height = 3
	}
	return height
}

// Init implements tea.Model.
//
// A seeded run still seeds the form from the flags before the first command, so
// the user can press enter to confirm what the flags proposed.
func (m *Model) Init() tea.Cmd {
	m.applySeed()
	return m.form.Init()
}

// applySeed copies flag values into the answers the active form binds.
func (m *Model) applySeed() {
	s := m.opts.Seed
	switch m.screen {
	case ScreenPaintText:
		if v := strings.TrimSpace(s.Text); v != "" {
			m.paint.Text = v
		}
		if v := strings.TrimSpace(s.Intensity); v != "" {
			m.paint.Intensity = v
		}
	case ScreenPaintPattern:
		if v := strings.TrimSpace(s.Pattern); v != "" {
			m.ptrn.Pattern = v
		}
	case ScreenSimulate:
		if v := strings.TrimSpace(s.Days); v != "" {
			m.sim.Days = v
		}
		if v := strings.TrimSpace(s.Profile); v != "" {
			m.sim.Profile = v
		}
		if v := strings.TrimSpace(s.Seed); v != "" {
			m.sim.Seed = v
		}
	case ScreenCompare:
		if v := strings.TrimSpace(s.Before); v != "" {
			m.cmp.Before = v
		}
		if v := strings.TrimSpace(s.After); v != "" {
			m.cmp.After = v
		}
	}
	// Rebuild so the seeded values are validated and any Select is re-seeded
	// against its own options.
	m.form = m.buildForm()
}

// --- commands ----------------------------------------------------------------------

// paintTextCmd runs the grid preview and the commit plan for a text paint.
//
// Two calls, not one: `--preview` prints only the grid, while `--dry-run` prints
// only the plan table and its totals. The user wants both, and the CLI has no
// single invocation that emits them together.
func (m *Model) paintTextCmd() tea.Cmd {
	text := strings.TrimSpace(m.paint.Text)
	intensity, _ := strconv.Atoi(strings.TrimSpace(m.paint.Intensity))
	if err := ValidatePaintText(text); err != nil {
		return failureCmd(err)
	}
	return m.paintCmd(
		cli.TextPreview(m.opts.Launcher, text, intensity),
		cli.TextPlan(m.opts.Launcher, text, intensity),
	)
}

// paintPatternCmd runs the preview and the plan for a built-in pattern.
func (m *Model) paintPatternCmd() tea.Cmd {
	pattern := strings.TrimSpace(m.ptrn.Pattern)
	if pattern == "" {
		return failureCmd(errors.New("no pattern is available; the CLI reported none"))
	}
	// A pattern the CLI does not offer is refused here rather than run. A huh Select
	// has already replaced the requested name with the first option, so without this
	// check a request for a pattern that no longer exists would quietly paint a
	// different one.
	if bad := strings.TrimSpace(m.ptrn.Unavailable); bad != "" {
		return failureCmd(fmt.Errorf("pattern %q is not one of this CLI's patterns (%s); choose one instead",
			bad, strings.Join(m.opts.Patterns, ", ")))
	}
	return m.paintCmd(
		cli.PatternPreview(m.opts.Launcher, pattern),
		cli.PatternPlan(m.opts.Launcher, pattern),
	)
}

// paintCmd runs a preview followed by a plan and folds both into one message.
//
// The preview is treated as the more important result: if only one of the two
// calls can succeed, the grid is what the user came for, so a plan failure does
// not discard a good grid.
func (m *Model) paintCmd(preview, plan cli.Invocation) tea.Cmd {
	exec := m.opts.Runner
	timeout := m.opts.Timeout
	return func() tea.Msg {
		msg := outputMsg{
			kind:     "paint",
			commands: []string{preview.CommandLine(), plan.CommandLine()},
		}
		if exec == nil {
			msg.err = errNoRunner
			return msg
		}
		gridOut, gridErr := exec.Run(preview, timeout)
		planOut, planErr := exec.Run(plan, timeout)
		msg.preview = gridOut
		msg.raw = planOut
		if gridErr != nil {
			msg.err = gridErr
		} else if planErr != nil {
			msg.err = planErr
		}
		return msg
	}
}

// simulateCmd runs `simulate --dry-run`.
func (m *Model) simulateCmd() tea.Cmd {
	if err := ValidateDays(m.sim.Days); err != nil {
		return failureCmd(err)
	}
	if err := ValidateOptionalSeed(m.sim.Seed); err != nil {
		return failureCmd(err)
	}
	// Same guard as the pattern picker: the CLI does validate profile names and
	// reports an unknown one, but the Select has already replaced the request with
	// the first option, so the rejection has to be caught before the substituted
	// name is sent.
	if bad := strings.TrimSpace(m.sim.Unavailable); bad != "" {
		return failureCmd(fmt.Errorf("profile %q is not one of this CLI's profiles (%s); choose one instead",
			bad, strings.Join(m.opts.Profiles, ", ")))
	}
	days, _ := strconv.Atoi(strings.TrimSpace(m.sim.Days))
	var seed *int
	if v := strings.TrimSpace(m.sim.Seed); v != "" {
		parsed, err := strconv.Atoi(v)
		if err != nil {
			return failureCmd(err)
		}
		seed = &parsed
	}
	inv, err := cli.SimulatePlan(m.opts.Launcher, cli.SimulateOptions{
		Days:    days,
		Profile: m.sim.Profile,
		Seed:    seed,
		DryRun:  true,
	})
	if err != nil {
		return failureCmd(err)
	}
	return m.execCmd(inv, "simulate")
}

// compareCmd runs `stats compare`.
func (m *Model) compareCmd() tea.Cmd {
	if err := ValidateSnapshotPath(m.cmp.Before); err != nil {
		return failureCmd(err)
	}
	if err := ValidateSnapshotPath(m.cmp.After); err != nil {
		return failureCmd(err)
	}
	inv, err := cli.Compare(m.opts.Launcher, cli.CompareOptions{
		Before: m.cmp.Before,
		After:  m.cmp.After,
	})
	if err != nil {
		return failureCmd(err)
	}
	return m.execCmd(inv, "compare")
}

// failureCmd turns a validation failure into an output message, so a bad
// combination is reported in the same panel as a failed subprocess.
func failureCmd(err error) tea.Cmd {
	return func() tea.Msg { return outputMsg{err: err} }
}

// execCmd builds the command that runs one invocation.
func (m *Model) execCmd(inv cli.Invocation, kind string) tea.Cmd {
	exec := m.opts.Runner
	timeout := m.opts.Timeout
	return func() tea.Msg {
		if exec == nil {
			return outputMsg{kind: kind, err: errNoRunner, commands: []string{inv.CommandLine()}}
		}
		out, err := exec.Run(inv, timeout)
		return outputMsg{kind: kind, raw: out, err: err, commands: []string{inv.CommandLine()}}
	}
}

// --- update ------------------------------------------------------------------------

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resize()
		// Re-render whatever is on screen at the new width.
		if m.last != nil && m.stage == stageResult {
			m.render(*m.last)
		} else if m.stage == stageRunning {
			m.viewport.SetContent(m.body)
		}
		return m, nil

	case outputMsg:
		// The screen that produced this result is the current one, so "run again" can
		// reopen the form the user actually submitted.
		m.resultScreen = m.screen
		m.stage = stageResult
		m.form = nil
		m.last = &msg
		m.render(msg)
		return m, nil

	case tea.KeyPressMsg:
		// Only key PRESSES are handled. bubbletea v2 also delivers key RELEASES,
		// and acting on both would fire every binding twice per keystroke.
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.stage == stageResult {
			return m.updateResult(msg)
		}
		m.notice = ""

		// Escape is intercepted here, BEFORE the form is given the message.
		//
		// huh's default keymap binds Quit to ctrl+c ONLY, and the one path that sets
		// StateAborted is key.Matches(msg, keymap.Quit), so a form never aborts on
		// escape. Delegating it therefore left the form installed: it held the focus
		// and silently swallowed every later keystroke, while the footer advertised
		// "esc back" and the user had no advertised way out.
		//
		// "esc back" is taken literally: the form is left behind for the menu, which
		// is what the unreachable StateAborted branch below always intended.
		if m.form != nil && msg.String() == "esc" {
			cmd := m.toMenu()
			// toMenu clears the notice, so set it after: the user needs to be told the
			// form was abandoned rather than submitted.
			m.notice = "Cancelled."
			return m, cmd
		}
	}

	if m.stage != stageForm || m.form == nil {
		return m, nil
	}

	updated, cmd := m.form.Update(msg)
	if form, ok := updated.(*huh.Form); ok {
		m.form = form
	}
	// Unreachable, and kept only as a backstop: ctrl+c returns tea.Quit at the top of
	// Update and escape is intercepted above, so huh never receives either key.
	// StateAborted is set solely by key.Matches(msg, keymap.Quit), which is bound to
	// ctrl+c alone. Do not rely on this branch for escape handling.
	if m.form.State == huh.StateAborted {
		return m, m.toMenu()
	}
	if m.form.State != huh.StateCompleted {
		return m, cmd
	}
	return m, m.advance()
}

// updateResult handles keys on the result stage.
//
// q and r are read before the viewport sees the message, so neither can be
// swallowed by the viewport's own bindings.
func (m *Model) updateResult(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		return m, m.toMenu()
	case "r":
		return m, m.runAgain()
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// advance reacts to a completed form.
func (m *Model) advance() tea.Cmd {
	switch m.screen {
	case ScreenPaintText:
		m.stage = stageRunning
		m.title = "Painting " + strconv.Quote(strings.TrimSpace(m.paint.Text))
		return tea.Batch(m.setRunning(), m.paintTextCmd())
	case ScreenPaintPattern:
		m.stage = stageRunning
		m.title = "Painting pattern " + m.ptrn.Pattern
		return tea.Batch(m.setRunning(), m.paintPatternCmd())
	case ScreenSimulate:
		m.stage = stageRunning
		m.title = "Simulating " + strings.TrimSpace(m.sim.Days) + " days"
		return tea.Batch(m.setRunning(), m.simulateCmd())
	case ScreenCompare:
		m.stage = stageRunning
		m.title = "Comparing snapshots"
		return tea.Batch(m.setRunning(), m.compareCmd())
	default:
		return m.menuAction()
	}
}

// setRunning paints a stable working body so the frame does not jump while a
// subprocess is in flight.
func (m *Model) setRunning() tea.Cmd {
	m.body = "Running the github-utils CLI…"
	m.viewport.SetContent(m.body)
	return nil
}

// menuAction routes the launcher selection.
func (m *Model) menuAction() tea.Cmd {
	switch m.menu.Choice {
	case ActionPaintText:
		m.screen = ScreenPaintText
		return m.setForm(m.buildForm())
	case ActionPaintPattern:
		m.screen = ScreenPaintPattern
		return m.setForm(m.buildForm())
	case ActionSimulate:
		m.screen = ScreenSimulate
		return m.setForm(m.buildForm())
	case ActionCompare:
		m.screen = ScreenCompare
		return m.setForm(m.buildForm())
	case ActionAbout:
		// About is purely local: it describes what this front-end is and which
		// command it will delegate to. Running a subprocess to say so would be
		// slower and no more informative.
		m.stage = stageResult
		m.form = nil
		m.last = nil
		m.resultScreen = ScreenMenu
		m.failure = nil
		m.title = "About this front-end"
		m.commands = []string{m.opts.Launcher.Describe("--help")}
		m.body = m.aboutBody()
		m.viewport.SetContent(m.body)
		m.viewport.GotoTop()
		return nil
	default:
		return nil
	}
}

// toMenu returns to the launcher.
func (m *Model) toMenu() tea.Cmd {
	m.screen = ScreenMenu
	m.stage = stageForm
	m.failure = nil
	m.notice = ""
	m.body = ""
	m.commands = nil
	m.last = nil
	return m.setForm(m.buildForm())
}

// runAgain reopens the form for the action that produced the current result.
func (m *Model) runAgain() tea.Cmd {
	m.screen = m.resultScreen
	m.stage = stageForm
	m.failure = nil
	return m.setForm(m.buildForm())
}

// --- rendering ---------------------------------------------------------------------

// render turns a raw subprocess result into the title, body and viewport content.
func (m *Model) render(msg outputMsg) {
	m.commands = msg.commands
	m.failure = msg.err
	m.notice = ""

	var top strings.Builder
	if p := parse.SummaryOf(msg.raw); p.HasAny() {
		top.WriteString(m.summaryLines(p))
		top.WriteString("\n\n")
	}

	// A grid preview is drawn with the token palette rather than passed through,
	// which is the whole point of this front-end: the same intensities the CLI
	// computed, sized and coloured for a terminal.
	if len(msg.preview) > 0 {
		if parsed, err := grid.ParsePreview(msg.preview); err != nil {
			// A parse failure is reported, never rendered as an empty grid: an
			// empty calendar would look like a legitimate result.
			top.WriteString(PaneError("Grid preview", err, m.paneWidth()))
			top.WriteString("\n\n")
		} else {
			top.WriteString(m.gridPanel(parsed))
			top.WriteString("\n\n")
		}
	}

	body := strings.TrimSpace(stripANSI(string(msg.raw)))

	m.body = strings.TrimSpace(top.String())
	m.viewport.SetHeight(m.viewportHeight())
	if m.failure != nil {
		pane := PaneError("The CLI reported a failure", m.failure, m.paneWidth())
		m.viewport.SetContent(pane + "\n\n" + body)
	} else if body != "" {
		m.viewport.SetContent(body)
	} else {
		m.viewport.SetContent("The CLI produced no output.")
	}
	m.viewport.GotoTop()
}

// summaryLines renders the totals the CLI printed.
func (m *Model) summaryLines(p parse.Summary) string {
	var lines []string
	if p.TotalCommits > 0 {
		lines = append(lines, Stat("Total estimated commits", strconv.Itoa(p.TotalCommits)))
	}
	if p.SimulatedCommits > 0 {
		lines = append(lines, Stat("Total commits", strconv.Itoa(p.SimulatedCommits)))
	}
	switch {
	case p.WindowDays > 0:
		lines = append(lines, Stat("Active days", strconv.Itoa(p.ActiveDays)+" / "+strconv.Itoa(p.WindowDays)))
	case p.ActiveDays > 0:
		lines = append(lines, Stat("Active days", strconv.Itoa(p.ActiveDays)))
	}
	if p.HasAvg {
		lines = append(lines, Stat("Average per active day", strconv.FormatFloat(p.AvgPerActiveDay, 'f', -1, 64)))
	}
	return strings.Join(lines, "\n")
}

// gridPanel renders the heatmap and its legend inside a pane.
func (m *Model) gridPanel(g *grid.Grid) string {
	content := innerWidth(m.paneWidth())
	heat, columns, truncated := Heatmap(g, content)
	if truncated > 0 {
		heat += "\n" + Legend() + "  " + Truncate(
			"terminal too narrow; "+strconv.Itoa(truncated)+" of 52 weeks hidden", content)
	} else {
		heat += "\n" + Legend()
	}
	title := "Contribution graph"
	if columns < grid.Cols {
		title = "Contribution graph (" + strconv.Itoa(columns) + "/52 weeks)"
	}
	return Pane(title, heat, m.paneWidth())
}

// aboutBody describes the front-end without running anything.
func (m *Model) aboutBody() string {
	var b strings.Builder
	b.WriteString("This is an additional way to run github-utils.\n\n")
	b.WriteString("It paints nothing itself. Every glyph, pattern, profile and commit\n")
	b.WriteString("estimate you see here is produced by the repository's own Python CLI,\n")
	b.WriteString("which this front-end runs — so a Go user and a Python user get identical\n")
	b.WriteString("results.\n\n")
	b.WriteString(Stat("Delegates to", m.opts.Launcher.Bin) + "\n")
	if len(m.opts.Launcher.Prefix) > 0 {
		b.WriteString(Stat("Prefix", strings.Join(m.opts.Launcher.Prefix, " ")) + "\n")
	}
	if v := strings.TrimSpace(m.opts.Version); v != "" {
		b.WriteString(Stat("CLI version", v) + "\n")
	}
	b.WriteString(Stat("Patterns found", strconv.Itoa(len(m.opts.Patterns))) + "\n")
	b.WriteString(Stat("Profiles found", strconv.Itoa(len(m.opts.Profiles))) + "\n\n")
	b.WriteString("Keys: enter select · esc back · q quit")
	return b.String()
}

// paneWidth is the total width a pane may occupy.
func (m *Model) paneWidth() int { return max(m.width-2, minPaneTotal) }

// View implements tea.Model.
func (m *Model) View() tea.View {
	var b strings.Builder

	header := Header(m.opts.Version, max(m.width, 20))
	b.WriteString(header)
	b.WriteString("\n\n")

	if m.stage == stageResult {
		if m.title != "" {
			b.WriteString(m.title + "\n\n")
		}
		if m.body != "" {
			b.WriteString(m.body)
			b.WriteString("\n\n")
		}
		// The viewport is ALWAYS drawn, not only when body is empty. Setting its
		// content without rendering it left the CLI's plan table unreachable and made
		// the pager keys appear broken, which is exactly the bug this guards against.
		b.WriteString(m.viewport.View())
	} else if m.stage == stageRunning {
		b.WriteString(m.body)
	} else if m.form != nil {
		b.WriteString(m.form.View())
	}

	if m.notice != "" {
		b.WriteString("\n\n" + m.notice)
	}
	if cmd := m.commandHint(); cmd != "" {
		b.WriteString("\n\n" + Footer(cmd, m.width))
	}
	b.WriteString("\n" + Footer(m.footerHint(), m.width))

	return tea.NewView(b.String())
}

// commandHint shows the command that produced the current frame, so the user can
// always reproduce it in a shell.
func (m *Model) commandHint() string {
	if len(m.commands) == 0 {
		return ""
	}
	return "$ " + strings.Join(m.commands, "\n$ ")
}

// footerHint is the key hint for the active stage.
func (m *Model) footerHint() string {
	if m.stage == stageResult {
		return "r run again · esc back to the menu · q quit · up/down scroll"
	}
	if m.stage == stageRunning {
		return "running… · ctrl+c quit"
	}
	return "enter select · shift+tab back · esc back · ctrl+c quit"
}

// stripANSI removes SGR colour sequences from passthrough text.
//
// The CLI colours only a terminal, so piped output normally has none; this exists
// for FORCE_COLOR, and to keep a captured colour code out of a pane that applies
// its own.
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
