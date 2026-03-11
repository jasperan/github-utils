"""Legacy bridge -- backward compatibility with the original utils.py CLI."""

from __future__ import annotations

from typing import Optional

import typer
from rich.console import Console

legacy_app = typer.Typer(help="Legacy mode: backward-compatible interface for old utils.py commands.")

console = Console()


@legacy_app.command("commit")
def commit(
    count: int = typer.Argument(..., help="Number of commits to create."),
    dry_run: bool = typer.Option(False, "--dry-run", help="Show plan without executing."),
) -> None:
    """Create a specified number of commits (legacy mode).

    Example: github-utils legacy commit 50000 --dry-run
    """
    if dry_run:
        console.print(f"[bold]Legacy mode (dry-run):[/bold] Would create {count} commits.")
        console.print()
        console.print("[dim]This is the legacy bridge for backward compatibility with utils.py.[/dim]")
        console.print("[dim]Consider using 'github-utils paint' or 'github-utils simulate' instead.[/dim]")
    else:
        console.print(f"[bold]Legacy mode:[/bold] Would create {count} commits.")
        console.print("[yellow]Execution not yet implemented.[/yellow]")
        console.print()
        console.print("[dim]Use --dry-run to preview. Consider migrating to the new commands:[/dim]")
        console.print("[dim]  github-utils paint   -- paint text/patterns[/dim]")
        console.print("[dim]  github-utils simulate -- realistic activity[/dim]")


@legacy_app.command("branch")
def branch(
    mode: str = typer.Argument(..., help="Branch mode (e.g. 'inf' for infinite)."),
    dry_run: bool = typer.Option(False, "--dry-run", help="Show plan without executing."),
) -> None:
    """Manage branches in legacy mode.

    Example: github-utils legacy branch inf --dry-run
    """
    if dry_run:
        console.print(f"[bold]Legacy mode (dry-run):[/bold] Branch operation mode='{mode}'.")
        console.print()
        console.print("[dim]This is the legacy bridge for backward compatibility with utils.py.[/dim]")
    else:
        console.print(f"[bold]Legacy mode:[/bold] Branch operation mode='{mode}'.")
        console.print("[yellow]Execution not yet implemented.[/yellow]")
