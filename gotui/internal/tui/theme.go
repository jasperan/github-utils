package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/jasperan/github-utils/gotui/internal/grid"
	"github.com/jasperan/github-utils/gotui/internal/huhstyle"
)

// minPaneWidth is the narrowest a panel's CONTENT area may render at.
//
// A terminal narrower than this is common in a split pane, and rendering below
// it produced borders narrower than their own title. Clamping is what keeps a
// narrow window from panicking on a negative repeat count.
const minPaneWidth = 20

// paneChrome is the columns a pane spends on its own frame: a 2-column border
// plus 2 columns of padding on each side.
//
// It exists because lipgloss Width(w) sets the block's TOTAL width, frame
// included, so its content area is only w - paneChrome. This is measured, not
// assumed: lipgloss.NewStyle().Border(RoundedBorder()).Padding(1, 2).Width(w)
// renders exactly w columns, leaving w - 6 for text. Subtracting the chrome
// twice makes every box wrap at the edge of the terminal.
const paneChrome = 6

// minPaneTotal is the narrowest a pane's TOTAL width may be, so its content area
// never drops below minPaneWidth.
const minPaneTotal = minPaneWidth + paneChrome

// innerWidth converts the total columns a pane occupies into the content width
// available inside its frame.
func innerWidth(total int) int {
	inner := total - paneChrome
	if inner < minPaneWidth {
		inner = minPaneWidth
	}
	return inner
}

// palette derives every colour from the theme. No literal is declared here.
//
// Reading the compiled styles back out of internal/huhstyle keeps ONE definition
// of each token: a colour written in this file would instantly diverge from the
// forms the user is looking at, and the canonical theme file is copied
// byte-for-byte precisely so it stays the single source of truth.
//
// The theme exposes eight of the fourteen tokens through its styles. The others
// (surface, highest, secondary, success, warning) are not painted by any form
// style, so this front-end does not use them rather than re-declaring them.
func palette() (primary, text, subtext, muted, danger, info lipgloss.Style) {
	styles := huhstyle.Styles()
	return lipgloss.NewStyle().Foreground(styles.Focused.Title.GetForeground()),
		lipgloss.NewStyle().Foreground(styles.Focused.Option.GetForeground()),
		lipgloss.NewStyle().Foreground(styles.Focused.Description.GetForeground()),
		lipgloss.NewStyle().Foreground(styles.Focused.TextInput.Placeholder.GetForeground()),
		lipgloss.NewStyle().Foreground(styles.Focused.ErrorMessage.GetForeground()),
		lipgloss.NewStyle().Foreground(styles.Focused.Directory.GetForeground())
}

// Pane renders a titled panel: rounded border, generous padding, primary title.
func Pane(title, body string, width int) string {
	primary, text, _, _, _, _ := palette()

	if width < minPaneTotal {
		width = minPaneTotal
	}
	heading := ""
	if title != "" {
		heading = primary.Bold(true).Render(title) + "\n"
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(width).
		Render(heading + text.Render(body))
}

// PaneError renders a failure panel. A failed CLI call is a normal state in this
// front-end, not a crash: the user gets the cause and a way onward.
func PaneError(title string, err error, width int) string {
	_, _, _, _, danger, _ := palette()
	message := "unknown error"
	if err != nil {
		message = err.Error()
	}
	return Pane(title, danger.Render(message), width)
}

// Header is the persistent identity bar.
func Header(version string, width int) string {
	primary, _, subtext, _, _, _ := palette()
	title := primary.Bold(true).Render("github-utils") + " " + subtext.Render("contribution graph studio")
	right := subtext.Render(strings.TrimSpace(version))
	gap := width - lipgloss.Width(title) - lipgloss.Width(right) - 4
	if gap < 1 {
		gap = 1
	}
	return title + strings.Repeat(" ", gap) + right
}

// Footer is the key-hint bar.
//
// It truncates rather than wrapping, so a narrow terminal cannot push the hint
// onto a second line and shift the whole frame.
func Footer(hint string, width int) string {
	_, _, subtext, _, _, _ := palette()
	if width > 2 && lipgloss.Width(hint) > width-2 {
		hint = Truncate(hint, width-2)
	}
	return subtext.Render(hint)
}

// Stat renders one "label: value" line for the readouts.
func Stat(label, value string) string {
	_, text, subtext, _, _, _ := palette()
	return subtext.Render(label+": ") + text.Render(value)
}

// Delta renders a signed change.
//
// The sign is the primary signal and colour is a secondary cue, so a reader who
// sees no colour still gets the meaning. No semantic token is claimed here: the
// theme exposes no success colour, and a rising contribution count is a number
// rather than an event.
func Delta(n int) string {
	primary, text, _, muted, _, _ := palette()
	switch {
	case n > 0:
		return primary.Render("+" + strconv.Itoa(n))
	case n < 0:
		return primary.Render(strconv.Itoa(n))
	default:
		return muted.Render(text.Render("0"))
	}
}

// Heat glyphs, low to high.
//
// Levels 1-4 use exactly the block characters the CLI's own preview uses, so the
// shape on screen is the shape the Python renderer produced. Level 0 is the one
// deliberate difference: upstream prints a space, which leaves the calendar
// illegible, so an empty cell is drawn as a faint dot.
const (
	glyphEmpty = '·'
	glyphLight = '░' // U+2591
	glyphMed   = '▒' // U+2592
	glyphDark  = '▓' // U+2593
	glyphFull  = '█' // U+2588
)

// heatGlyph is the glyph for an intensity level.
var heatGlyph = [5]rune{glyphEmpty, glyphLight, glyphMed, glyphDark, glyphFull}

// heatStyle returns the colour for an intensity level.
//
// The ramp walks the theme's neutrals into its two accents, so weight grows with
// value on a terminal with no colour at all: muted, muted, subtext, info,
// primary. That is two accents at most in one view, which the design rules allow.
func heatStyle(level int) lipgloss.Style {
	primary, _, subtext, muted, _, info := palette()
	switch {
	case level <= 0:
		return muted
	case level == 1:
		return muted
	case level == 2:
		return subtext
	case level == 3:
		return info
	default:
		return primary.Bold(true)
	}
}

// Heatmap renders the parsed grid as a coloured calendar.
//
// width is the TOTAL columns allocated to the map. A grid is 52 columns wide, and
// a narrower terminal shows fewer whole columns rather than wrapping or panicking,
// with the caller told via truncated so it can say so on screen.
func Heatmap(g *grid.Grid, width int) (rendered string, columns, truncated int) {
	columns = grid.Cols
	if width > 0 && width < grid.Cols {
		columns = max(width, 1)
	}
	truncated = grid.Cols - columns

	var out strings.Builder
	for row := 0; row < grid.Rows; row++ {
		for col := 0; col < columns; col++ {
			level := g.At(row, col)
			out.WriteString(heatStyle(level).Render(string(heatGlyph[level])))
		}
		if row < grid.Rows-1 {
			out.WriteString("\n")
		}
	}
	return out.String(), columns, truncated
}

// Legend explains the intensity scale under the map.
func Legend() string {
	_, _, subtext, muted, _, _ := palette()
	var parts []string
	for level := 0; level < 5; level++ {
		parts = append(parts, heatStyle(level).Render(string(heatGlyph[level])))
	}
	return muted.Render("empty ") + strings.Join(parts, "") + subtext.Render(" dense")
}

// Pad right-pads to n columns.
func Pad(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// Truncate shortens s to at most n runes, adding an ellipsis when it cuts.
func Truncate(s string, n int) string {
	if n <= 1 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// max returns the larger of two ints.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
