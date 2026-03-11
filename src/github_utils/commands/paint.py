"""Paint command -- render text, patterns, or files onto the contribution graph."""

from __future__ import annotations

from pathlib import Path
from typing import Optional

import typer
from rich.console import Console
from rich.table import Table

from github_utils.painter.font import render_text
from github_utils.painter.grid import ContributionGrid
from github_utils.painter.patterns import list_builtin_patterns, load_pattern
from github_utils.painter.preview import render_grid_preview

console = Console()


def _resolve_grid(
    text: Optional[str],
    pattern: Optional[str],
    file: Optional[Path],
    intensity: int,
) -> ContributionGrid:
    """Resolve the grid from the given input source."""
    if file is not None:
        return load_pattern(str(file))
    if pattern is not None:
        return load_pattern(pattern)
    if text is not None:
        return render_text(text, intensity=intensity)
    # Fallback: empty grid (shouldn't reach here due to validation)
    return ContributionGrid.empty()


def _render_preview(grid: ContributionGrid) -> None:
    """Render a terminal grid preview using block characters and Rich colors."""
    render_grid_preview(grid, console=console)


def _render_commit_plan(grid: ContributionGrid) -> None:
    """Show the commit plan as a Rich table."""
    cells = grid.nonzero_cells()
    if not cells:
        console.print("[yellow]No commits needed -- grid is empty.[/yellow]")
        return

    table = Table(title="Commit Plan")
    table.add_column("Date", style="cyan")
    table.add_column("Intensity", justify="center")
    table.add_column("Commits (est.)", justify="right", style="green")

    total = 0
    for row, col, intensity in cells:
        d = grid.cell_to_date(row, col)
        # Estimate commits: intensity level directly (actual calibration happens at execution)
        est_commits = intensity
        total += est_commits
        table.add_row(str(d), str(intensity), str(est_commits))

    console.print(table)
    console.print(f"\n[bold]Total estimated commits:[/bold] {total}")
    console.print(f"[bold]Active days:[/bold] {len(cells)}")


def paint_command(
    text: Optional[str] = typer.Argument(None, help="Text to paint on the contribution graph."),
    pattern: Optional[str] = typer.Option(None, "--pattern", "-p", help="Named pattern (e.g. wave, heart, checkerboard)."),
    file: Optional[Path] = typer.Option(None, "--file", "-f", help="Path to a YAML pattern file."),
    intensity: int = typer.Option(4, "--intensity", "-i", min=1, max=4, help="Intensity level for text mode (1-4)."),
    preview: bool = typer.Option(False, "--preview", help="Show terminal grid preview only."),
    dry_run: bool = typer.Option(False, "--dry-run", help="Show commit plan without executing."),
) -> None:
    """Paint text or patterns on your GitHub contribution graph."""
    # Validate: need at least one input source
    sources = sum(x is not None for x in [text, pattern, file])
    if sources == 0:
        console.print("[red]Error:[/red] Provide text argument, --pattern, or --file.")
        console.print(f"\nAvailable patterns: {', '.join(list_builtin_patterns()) or 'none'}")
        raise typer.Exit(code=1)
    if sources > 1:
        console.print("[red]Error:[/red] Provide only one of: text argument, --pattern, or --file.")
        raise typer.Exit(code=1)

    try:
        grid = _resolve_grid(text, pattern, file, intensity)
    except FileNotFoundError as exc:
        console.print(f"[red]Error:[/red] {exc}")
        raise typer.Exit(code=1)

    if preview:
        _render_preview(grid)

    if dry_run:
        _render_commit_plan(grid)

    if not preview and not dry_run:
        _render_preview(grid)
        console.print("[yellow]Execution not yet implemented.[/yellow] Use --dry-run to see the commit plan.")
