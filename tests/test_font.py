from github_utils.painter.font import PixelFont, render_text
from github_utils.painter.grid import ContributionGrid


def test_font_has_alphanumeric():
    font = PixelFont()
    for ch in "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789":
        assert ch in font.glyphs, f"Missing glyph for {ch}"


def test_glyph_dimensions():
    font = PixelFont()
    for ch, glyph in font.glyphs.items():
        assert len(glyph) == 7, f"Glyph {ch} has {len(glyph)} rows, expected 7"
        for row in glyph:
            assert len(row) == 5, f"Glyph {ch} has row width {len(row)}, expected 5"


def test_render_single_char():
    grid = render_text("A", intensity=4)
    cells = grid.nonzero_cells()
    assert len(cells) > 0
    for _, _, intensity in cells:
        assert intensity == 4


def test_render_two_chars():
    grid = render_text("AB", intensity=3)
    cells = grid.nonzero_cells()
    cols_used = {c for _, c, _ in cells}
    assert 5 not in cols_used  # gap column between A and B


def test_render_max_length():
    """Max ~8 chars fit in 52 columns (5 wide + 1 gap each = 6 per char, 8*6-1=47)."""
    grid = render_text("ABCDEFGH", intensity=2)
    cells = grid.nonzero_cells()
    assert len(cells) > 0
    max_col = max(c for _, c, _ in cells)
    assert max_col < 52


def test_render_truncates_overflow():
    """Characters that would exceed column 51 are dropped."""
    grid = render_text("ABCDEFGHIJKLM", intensity=2)
    cells = grid.nonzero_cells()
    for _, col, _ in cells:
        assert col < 52
