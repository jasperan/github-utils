"""Grid preview renderer -- reusable terminal visualisation of a ContributionGrid.

Provides block-character rendering with Rich colour support.  Used by both
the ``paint`` and ``stats`` CLI commands.
"""

from __future__ import annotations

from rich.console import Console
from rich.text import Text

from github_utils.painter.grid import ContributionGrid

# Block characters mapping intensity 0-4 to visual weight.
INTENSITY_CHARS = [" ", "\u2591", "\u2592", "\u2593", "\u2588"]  # " ", "░", "▒", "▓", "█"

# Rich style strings for each intensity level.
INTENSITY_COLORS = ["dim", "green", "bright_green", "bold green", "bold bright_green"]


def render_grid_preview(
    grid: ContributionGrid,
    console: Console | None = None,
) -> str:
    """Render a coloured terminal preview of the grid.

    Parameters
    ----------
    grid:
        The contribution grid to render.
    console:
        Optional Rich console.  When provided the preview is printed with
        colour markup; when ``None`` the function only returns the plain-text
        representation.

    Returns
    -------
    str
        A plain-text string (one line per row, joined by newlines) using the
        block characters from ``INTENSITY_CHARS``.
    """
    lines: list[str] = []
    for row in range(grid.rows):
        line = ""
        for col in range(grid.cols):
            v = grid.get(row, col)
            line += INTENSITY_CHARS[v]
        lines.append(line)

    if console is not None:
        console.print()
        console.print("[bold]Grid Preview[/bold]")
        console.print()
        for row in range(grid.rows):
            rich_line = Text()
            for col in range(grid.cols):
                v = grid.get(row, col)
                rich_line.append(INTENSITY_CHARS[v], style=INTENSITY_COLORS[v])
            console.print(rich_line)
        console.print()

    return "\n".join(lines)
