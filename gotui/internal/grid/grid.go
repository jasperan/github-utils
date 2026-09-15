// Package grid parses the contribution grid out of the CLI's own preview output.
//
// The grid is 7 rows by 52 columns of intensity 0-4, and the front-end does not
// compute it: `github-utils paint <TEXT> --preview` renders it, and this package
// reads those glyphs back. A glyph renderer, a font, and the pattern loader all
// stay in Python, so a Go user and a Python user see the same shape.
//
// The glyph table mirrors INTENSITY_CHARS in src/github_utils/painter/preview.py,
// which is the only place the mapping is defined upstream.
package grid

import (
	"errors"
	"fmt"
	"strings"
)

// Rows and Cols are ContributionGrid.ROWS and ContributionGrid.COLS.
const (
	Rows = 7
	Cols = 52
)

// PreviewHeader is the title the CLI prints above the grid.
const PreviewHeader = "Grid Preview"

// Intensity glyphs, low to high, as preview.py declares them.
const (
	glyphEmpty = ' '
	glyphLight = '░' // U+2591
	glyphMed   = '▒' // U+2592
	glyphDark  = '▓' // U+2593
	glyphFull  = '█' // U+2588
)

// intensityOf maps a preview glyph to its intensity level.
var intensityOf = map[rune]int{
	glyphEmpty: 0,
	glyphLight: 1,
	glyphMed:   2,
	glyphDark:  3,
	glyphFull:  4,
}

// ErrNoPreview is returned when the output holds no grid.
var ErrNoPreview = errors.New("the CLI output holds no grid preview")

// Grid is a parsed contribution grid.
type Grid struct {
	cells [Rows][Cols]int
}

// At returns the intensity at a cell, clamped so an out-of-range index cannot
// panic a renderer that is drawing at a narrow width.
func (g *Grid) At(row, col int) int {
	if g == nil || row < 0 || row >= Rows || col < 0 || col >= Cols {
		return 0
	}
	return g.cells[row][col]
}

// Counts returns how many cells hold each intensity, indexed 0-4.
//
// A renderer uses this to size a legend, and a test uses it to prove a parsed
// shape matches a rendered one.
func (g *Grid) Counts() [5]int {
	var counts [5]int
	if g == nil {
		return counts
	}
	for r := 0; r < Rows; r++ {
		for c := 0; c < Cols; c++ {
			counts[g.cells[r][c]]++
		}
	}
	return counts
}

// NonZero is the number of cells with any intensity, which matches the CLI's
// "Active days" figure for a painted grid.
func (g *Grid) NonZero() int {
	counts := g.Counts()
	active := 0
	for level := 1; level < len(counts); level++ {
		active += counts[level]
	}
	return active
}

// Total is the sum of every cell's intensity.
//
// The CLI estimates one commit per intensity level
// (commands/paint.py: `est_commits = intensity`), so this is the front-end's
// independent cross-check of the "Total estimated commits" line the CLI prints
// from the same grid. A mismatch means the grid and the table disagree, which is
// worth surfacing rather than hiding.
func (g *Grid) Total() int {
	counts := g.Counts()
	total := 0
	for level, n := range counts {
		total += level * n
	}
	return total
}

// ParsePreview reads the grid from `paint --preview` output.
//
// The layout is a "Grid Preview" title, then one blank line, then exactly seven
// rows. Each row is padded with spaces by Rich to the console width, and a space
// is also the glyph for intensity 0, so trailing padding is indistinguishable
// from empty cells and simply becomes intensity 0.
//
// Only one blank line after the title is skipped, deliberately. Skipping blank
// lines until the first non-blank one would corrupt a grid whose top rows are
// empty, which is the normal case for text painted towards the bottom.
func ParsePreview(out []byte) (*Grid, error) {
	text := stripANSI(string(out))
	lines := strings.Split(text, "\n")

	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == PreviewHeader {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, ErrNoPreview
	}

	first := start + 1
	if first < len(lines) && strings.TrimSpace(lines[first]) == "" {
		first++
	}
	if first+Rows > len(lines) {
		return nil, fmt.Errorf("%w: found %d of %d grid rows", ErrNoPreview, len(lines)-first, Rows)
	}

	g := &Grid{}
	for r := 0; r < Rows; r++ {
		values, err := parseRow(lines[first+r])
		if err != nil {
			return nil, fmt.Errorf("grid row %d: %w", r, err)
		}
		g.cells[r] = values
	}
	return g, nil
}

// parseRow converts one rendered row into 52 intensities.
//
// A short row is padded with zeroes and a long one truncated, so a narrower
// console cannot fail the parse. An unrecognised glyph IS an error: silently
// reading it as empty would show a wrong grid that looks plausible, which is
// worse than reporting that the CLI's output changed.
func parseRow(line string) ([Cols]int, error) {
	var out [Cols]int
	col := 0
	for _, r := range line {
		if col >= Cols {
			break
		}
		v, ok := intensityOf[r]
		if !ok {
			return out, fmt.Errorf("unrecognised glyph %q (U+%04X); the CLI renderer's glyph table changed", r, r)
		}
		out[col] = v
		col++
	}
	return out, nil
}

// stripANSI removes SGR colour sequences. Piped output normally has none,
// because Rich only colours a terminal, but FORCE_COLOR would produce them.
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
