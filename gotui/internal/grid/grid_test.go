package grid

import (
	"errors"
	"strings"
	"testing"
)

// wavePreview is the real output of `github-utils paint --pattern wave --preview`,
// captured from the CLI. The trailing spaces Rich pads each row with are omitted
// here, which also exercises the short-row padding path: a space and a missing
// character are both intensity 0.
const wavePreview = `
Grid Preview

█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░
▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░
▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░
░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒
 ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓
░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█
▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓▒░ ░▒▓█▓
`

// TestParsePreviewReadsARealGrid pins the parser against output captured from the
// CLI, so the glyph table cannot drift from preview.py unnoticed.
func TestParsePreviewReadsARealGrid(t *testing.T) {
	g, err := ParsePreview([]byte(wavePreview))
	if err != nil {
		t.Fatalf("ParsePreview: %v", err)
	}

	// Row 0 of the wave begins with the descending ramp, so its first four cells
	// are full, dark, medium and light.
	wantRow0 := []int{4, 3, 2, 1, 0}
	for col, want := range wantRow0 {
		if got := g.At(0, col); got != want {
			t.Errorf("At(0, %d) = %d, want %d", col, got, want)
		}
	}
	// Row 6 is the same ramp shifted, so its first cell is a medium block.
	if got := g.At(6, 0); got != 2 {
		t.Errorf("At(6, 0) = %d, want 2", got)
	}

	counts := g.Counts()
	if counts[0] == 0 || counts[4] == 0 {
		t.Errorf("counts = %v, want every level present in a wave", counts)
	}
	if total := g.NonZero() + 0; total != Rows*Cols-counts[0] {
		t.Errorf("NonZero = %d, want %d", total, Rows*Cols-counts[0])
	}
	if g.Total() <= 0 {
		t.Errorf("Total = %d, want a positive sum", g.Total())
	}
}

// TestParsePreviewKeepsLeadingEmptyRows is the bug this parser is shaped around.
//
// A grid whose top rows are empty is normal: text painted low on the calendar
// leaves the first rows blank. A parser that skipped blank lines to find the grid
// would silently drop those rows and shift the whole picture up by one or more,
// which looks plausible and is wrong.
func TestParsePreviewKeepsLeadingEmptyRows(t *testing.T) {
	blank := strings.Repeat(" ", Cols)
	full := strings.Repeat(string(glyphFull), Cols)
	fixture := "Grid Preview\n\n" + blank + "\n" + full + "\n" +
		blank + "\n" + blank + "\n" + blank + "\n" + blank + "\n" + blank + "\n"

	g, err := ParsePreview([]byte(fixture))
	if err != nil {
		t.Fatalf("ParsePreview: %v", err)
	}
	for col := 0; col < Cols; col++ {
		if got := g.At(0, col); got != 0 {
			t.Fatalf("At(0, %d) = %d, want 0; a blank leading row was dropped", col, got)
		}
		if got := g.At(1, col); got != 4 {
			t.Fatalf("At(1, %d) = %d, want 4; the grid shifted because a blank row was skipped", col, got)
		}
	}
}

// TestParsePreviewRejectsAnUnknownGlyph is the anti-silent-wrongness guard.
//
// Reading an unrecognised glyph as empty would render a plausible but wrong
// calendar. If the CLI's renderer changes its glyph table, the parse must say so.
func TestParsePreviewRejectsAnUnknownGlyph(t *testing.T) {
	bad := strings.Repeat("X", Cols)
	fixture := "Grid Preview\n\n" + bad + "\n" + strings.Repeat("\n", Rows-1)

	_, err := ParsePreview([]byte(fixture))
	if err == nil {
		t.Fatal("ParsePreview accepted an unrecognised glyph")
	}
	if !strings.Contains(err.Error(), "unrecognised glyph") {
		t.Errorf("error = %q, want it to name the unknown glyph", err)
	}
}

// TestParsePreviewRejectsMissingHeader keeps a non-preview output from being
// mistaken for an empty grid.
func TestParsePreviewRejectsMissingHeader(t *testing.T) {
	_, err := ParsePreview([]byte("Commit Plan\n┏━━━┓\n"))
	if !errors.Is(err, ErrNoPreview) {
		t.Fatalf("error = %v, want ErrNoPreview", err)
	}
}

// TestParsePreviewRejectsTruncatedOutput catches a wrapped or clipped capture,
// which would otherwise yield a ragged grid.
func TestParsePreviewRejectsTruncatedOutput(t *testing.T) {
	fixture := "Grid Preview\n\n" + strings.Repeat(string(glyphFull)+"\n", 3)
	_, err := ParsePreview([]byte(fixture))
	if !errors.Is(err, ErrNoPreview) {
		t.Fatalf("error = %v, want ErrNoPreview for only 3 of 7 rows", err)
	}
}

// TestParsePreviewPadsShortAndClampsLongRows keeps a differently sized capture
// usable: a narrower console must not fail the parse, and a wider one must not
// overflow the array.
func TestParsePreviewPadsShortAndClampsLongRows(t *testing.T) {
	short := strings.Repeat(string(glyphFull), 10)
	long := strings.Repeat(string(glyphFull), Cols+25)
	fixture := "Grid Preview\n\n" + short + "\n" + long + "\n" + short + "\n" +
		short + "\n" + short + "\n" + short + "\n" + short + "\n"

	g, err := ParsePreview([]byte(fixture))
	if err != nil {
		t.Fatalf("ParsePreview: %v", err)
	}
	if got := g.At(0, 9); got != 4 {
		t.Errorf("At(0, 9) = %d, want 4", got)
	}
	if got := g.At(0, 10); got != 0 {
		t.Errorf("At(0, 10) = %d, want 0 (padded)", got)
	}
	if got := g.At(1, Cols-1); got != 4 {
		t.Errorf("At(1, %d) = %d, want 4", Cols-1, got)
	}
}

// TestParsePreviewStripsANSI keeps a forced-colour capture parseable.
func TestParsePreviewStripsANSI(t *testing.T) {
	row := "\x1b[38;2;137;180;250m" + strings.Repeat(string(glyphFull), Cols) + "\x1b[m"
	fixture := "Grid Preview\n\n" + row + "\n" + strings.Repeat(row+"\n", Rows-1)

	g, err := ParsePreview([]byte(fixture))
	if err != nil {
		t.Fatalf("ParsePreview: %v", err)
	}
	if got := g.At(0, 0); got != 4 {
		t.Errorf("At(0, 0) = %d, want 4", got)
	}
}

// TestAtClampsOutOfRange keeps a renderer that is drawing at a narrow width from
// panicking on an out-of-range index.
func TestAtClampsOutOfRange(t *testing.T) {
	var g *Grid
	if got := g.At(0, 0); got != 0 {
		t.Errorf("nil grid At = %d, want 0", got)
	}
	g = &Grid{}
	for _, c := range []struct{ row, col int }{{-1, 0}, {0, -1}, {Rows, 0}, {0, Cols}, {99, 99}} {
		if got := g.At(c.row, c.col); got != 0 {
			t.Errorf("At(%d, %d) = %d, want 0", c.row, c.col, got)
		}
	}
}

// TestCountsCoversEveryCell guards the totals the renderer relies on.
func TestCountsCoversEveryCell(t *testing.T) {
	g := &Grid{}
	total := 0
	for _, n := range g.Counts() {
		total += n
	}
	if total != Rows*Cols {
		t.Errorf("counts cover %d cells, want %d", total, Rows*Cols)
	}
}
