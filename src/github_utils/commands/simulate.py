"""Simulate command -- generate realistic activity plans using profile engines."""

from __future__ import annotations

from datetime import date, timedelta
from typing import Optional

import typer
from rich.console import Console
from rich.table import Table

from github_utils.realism.profile import ProfileEngine

simulate_app = typer.Typer(help="Simulate realistic GitHub activity patterns.")

console = Console()

_PROFILE_CHOICES = ["default", "night-owl", "weekend-warrior", "9-to-5", "open-source-maintainer"]


def _commit_color(count: int) -> str:
    """Return a Rich color string based on commit count."""
    if count == 0:
        return "dim"
    if count <= 2:
        return "green"
    if count <= 5:
        return "bright_green"
    if count <= 10:
        return "bold green"
    return "bold bright_green"


@simulate_app.callback(invoke_without_command=True)
def simulate(
    ctx: typer.Context,
    days: int = typer.Option(90, "--days", "-d", min=1, help="Number of days to simulate."),
    profile: str = typer.Option("default", "--profile", "-p", help=f"Activity profile ({', '.join(_PROFILE_CHOICES)})."),
    seed: Optional[int] = typer.Option(None, "--seed", "-s", help="Random seed for reproducibility."),
    dry_run: bool = typer.Option(False, "--dry-run", help="Show plan without executing."),
) -> None:
    """Simulate realistic GitHub activity patterns."""
    # Validate profile name
    if profile not in _PROFILE_CHOICES:
        console.print(f"[red]Error:[/red] Unknown profile '{profile}'.")
        console.print(f"Available profiles: {', '.join(_PROFILE_CHOICES)}")
        raise typer.Exit(code=1)

    try:
        engine = ProfileEngine.from_name(profile, seed=seed)
    except ValueError as exc:
        console.print(f"[red]Error:[/red] {exc}")
        raise typer.Exit(code=1)

    end_date = date.today()
    start_date = end_date - timedelta(days=days - 1)
    plan = engine.generate_plan(start_date, end_date)

    if dry_run:
        _show_plan(plan, profile)
    else:
        _show_plan(plan, profile)
        console.print()
        console.print("[yellow]Execution not yet implemented.[/yellow] Use --dry-run to see the plan.")


def _show_plan(plan, profile: str) -> None:
    """Display the simulation plan as a Rich table with summary."""
    table = Table(title=f"Simulation Plan (profile: {profile})")
    table.add_column("Date", style="cyan")
    table.add_column("Day", style="dim")
    table.add_column("Commits", justify="right")

    total_commits = 0
    active_days = 0

    for day_plan in plan:
        count = day_plan.commit_count
        total_commits += count
        if count > 0:
            active_days += 1
        color = _commit_color(count)
        day_name = day_plan.date.strftime("%a")
        table.add_row(str(day_plan.date), day_name, f"[{color}]{count}[/{color}]")

    console.print(table)
    console.print()

    avg = total_commits / active_days if active_days > 0 else 0
    console.print(f"[bold]Total commits:[/bold] {total_commits}")
    console.print(f"[bold]Active days:[/bold] {active_days} / {len(plan)}")
    console.print(f"[bold]Average per active day:[/bold] {avg:.1f}")
