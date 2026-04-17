"""Progress tracking state for long-running painting operations.

``ProgressState`` is a small mutable data bag that tracks commit/push counts,
throughput, and ETA so callers can render their own progress UI.
"""
from __future__ import annotations

import time
from dataclasses import dataclass, field


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
