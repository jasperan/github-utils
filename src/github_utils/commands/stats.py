"""Stats command -- snapshot and compare contribution analytics."""

from __future__ import annotations

from pathlib import Path

import typer
from rich.console import Console
from rich.table import Table

from github_utils.analytics.snapshot import Snapshot, compare_snapshots

stats_app = typer.Typer(help="Contribution analytics: snapshots and comparisons.")

console = Console()


@stats_app.command("snapshot")
def snapshot() -> None:
    """Take a point-in-time snapshot of your contribution statistics.

    Requires a GitHub API token (set via GITHUB_TOKEN or config file).
    """
    console.print("[yellow]Snapshot requires a GitHub API token.[/yellow]")
    console.print()
    console.print("Set your token via one of:")
    console.print("  - Environment variable: [bold]GITHUB_TOKEN[/bold]")
    console.print("  - Config file: [bold]~/.github-utils.yaml[/bold] (token field)")
    console.print("  - GitHub CLI: [bold]gh auth login[/bold]")


@stats_app.command("compare")
def compare(
    before: Path = typer.Option(..., "--before", "-b", help="Path to the 'before' snapshot JSON file."),
    after: Path = typer.Option(..., "--after", "-a", help="Path to the 'after' snapshot JSON file."),
) -> None:
    """Compare two snapshots and show the deltas."""
    if not before.exists():
        console.print(f"[red]Error:[/red] File not found: {before}")
        raise typer.Exit(code=1)
    if not after.exists():
        console.print(f"[red]Error:[/red] File not found: {after}")
        raise typer.Exit(code=1)

    try:
        snap_before = Snapshot.load(before)
        snap_after = Snapshot.load(after)
    except Exception as exc:
        console.print(f"[red]Error loading snapshots:[/red] {exc}")
        raise typer.Exit(code=1)

    deltas = compare_snapshots(snap_before, snap_after)

    table = Table(title="Snapshot Comparison")
    table.add_column("Metric", style="bold")
    table.add_column("Before", justify="right", style="cyan")
    table.add_column("After", justify="right", style="cyan")
    table.add_column("Delta", justify="right")

    fields = {
        "total_contributions": "Total Contributions",
        "current_streak": "Current Streak",
        "longest_streak": "Longest Streak",
        "active_days": "Active Days",
    }

    for key, label in fields.items():
        before_val = getattr(snap_before, key)
        after_val = getattr(snap_after, key)
        delta = deltas[key]
        if delta > 0:
            delta_str = f"[green]+{delta}[/green]"
        elif delta < 0:
            delta_str = f"[red]{delta}[/red]"
        else:
            delta_str = "[dim]0[/dim]"
        table.add_row(label, str(before_val), str(after_val), delta_str)

    console.print(table)
