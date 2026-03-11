"""Live progress monitoring for long-running painting operations.

Provides a ``ProgressState`` data model for tracking commit/push counts,
throughput, and ETA, plus a ``LiveMonitor`` wrapper around Rich's Live
display for real-time terminal feedback.
"""
from __future__ import annotations

import time
from dataclasses import dataclass, field

from rich.live import Live
from rich.table import Table


@dataclass
class ProgressState:
    """Mutable state bag that tracks painting progress.

    All mutation goes through ``record_commit`` / ``record_push`` so
    throughput and ETA stay consistent.
    """

    total_commits: int
    completed: int = 0
    pushes: int = 0
    _start_time: float = field(default_factory=time.monotonic, repr=False)

    # -- mutations -----------------------------------------------------------

    def record_commit(self) -> None:
        """Mark one commit as completed."""
        self.completed += 1

    def record_push(self) -> None:
        """Mark one push as completed."""
        self.pushes += 1

    # -- derived properties --------------------------------------------------

    @property
    def percentage(self) -> float:
        """Completion percentage (0.0 .. 100.0)."""
        if self.total_commits == 0:
            return 100.0
        return (self.completed / self.total_commits) * 100.0

    @property
    def elapsed(self) -> float:
        """Seconds since this state was created."""
        return time.monotonic() - self._start_time

    @property
    def throughput(self) -> float:
        """Commits per second since tracking started."""
        elapsed = self.elapsed
        if elapsed == 0 or self.completed == 0:
            return 0.0
        return self.completed / elapsed

    @property
    def eta_seconds(self) -> float | None:
        """Estimated seconds remaining, or *None* if unknown."""
        rate = self.throughput
        if rate == 0:
            return None
        remaining = self.total_commits - self.completed
        return remaining / rate


class LiveMonitor:
    """Rich-based terminal display that renders ``ProgressState`` live.

    Usage::

        state = ProgressState(total_commits=500)
        with LiveMonitor(state) as monitor:
            for day in plan:
                create_commits(day)
                state.record_commit()
                monitor.refresh()
    """

    def __init__(self, state: ProgressState) -> None:
        self.state = state
        self._live: Live | None = None

    # -- context manager -----------------------------------------------------

    def __enter__(self) -> LiveMonitor:
        self._live = Live(self._build_table(), refresh_per_second=4)
        self._live.__enter__()
        return self

    def __exit__(self, *exc_info: object) -> None:
        if self._live is not None:
            self._live.__exit__(*exc_info)
            self._live = None

    # -- public API ----------------------------------------------------------

    def refresh(self) -> None:
        """Re-render the live display with current state."""
        if self._live is not None:
            self._live.update(self._build_table())

    # -- internals -----------------------------------------------------------

    def _build_table(self) -> Table:
        s = self.state
        table = Table(title="Painting Progress", show_header=False)
        table.add_column("Metric", style="cyan")
        table.add_column("Value", style="bold")

        table.add_row("Commits", f"{s.completed} / {s.total_commits}")
        table.add_row("Pushes", str(s.pushes))
        table.add_row("Progress", f"{s.percentage:.1f}%")
        table.add_row("Throughput", f"{s.throughput:.2f} commits/s")

        eta = s.eta_seconds
        if eta is not None:
            mins, secs = divmod(int(eta), 60)
            table.add_row("ETA", f"{mins}m {secs}s")
        else:
            table.add_row("ETA", "calculating...")

        return table
