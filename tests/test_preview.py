from io import StringIO

from rich.console import Console

from github_utils.painter.grid import ContributionGrid
from github_utils.painter.preview import (
    INTENSITY_CHARS,
    INTENSITY_COLORS,
    render_grid_preview,
)


def test_preview_empty_grid():
    grid = ContributionGrid.empty()
    text = render_grid_preview(grid)
    assert len(text.split("\n")) == 7
    # Every character should be a space or newline
    assert all(c == " " or c == "\n" for c in text)


def test_preview_with_content():
    grid = ContributionGrid.empty()
    grid.set(0, 0, 4)
    grid.set(3, 25, 2)
    text = render_grid_preview(grid)
    assert "\u2588" in text  # "█"
    assert "\u2592" in text  # "▒"


def test_preview_line_length():
    grid = ContributionGrid.empty()
    text = render_grid_preview(grid)
    for line in text.split("\n"):
        assert len(line) == grid.cols


def test_preview_with_console():
    grid = ContributionGrid.empty()
    grid.set(0, 0, 4)
    buf = StringIO()
    con = Console(file=buf, force_terminal=True, width=120)
    text = render_grid_preview(grid, console=con)
    output = buf.getvalue()
    # Console output should contain the "Grid Preview" header
    assert "Grid Preview" in output
    # Plain text return should still work
    assert "\u2588" in text


def test_preview_all_intensities():
    grid = ContributionGrid.empty()
    for v in range(5):
        grid.set(v, 0, v)
    text = render_grid_preview(grid)
    for v in range(5):
        assert INTENSITY_CHARS[v] in text


def test_intensity_constants_length():
    assert len(INTENSITY_CHARS) == 5
    assert len(INTENSITY_COLORS) == 5
