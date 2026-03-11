"""Painter orchestrator: converts a ContributionGrid into a concrete commit plan."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date
from typing import List, Tuple

from github_utils.api.client import ContributionData
from github_utils.painter.grid import ContributionGrid


@dataclass
class PaintPlan:
    """A concrete plan mapping dates to the number of commits needed."""

    day_commits: List[Tuple[date, int]]

    @property
    def total_commits(self) -> int:
        return sum(count for _, count in self.day_commits)


def create_paint_plan(
    grid: ContributionGrid, calibration: ContributionData
) -> PaintPlan:
    """Convert a grid with intensity values into a commit plan using calibration data.

    For each non-zero cell in the grid, the corresponding calendar date is
    resolved and the calibration data determines how many commits are needed
    to reach that intensity level on the GitHub contribution graph.

    When multiple cells map to the same date (shouldn't normally happen in a
    well-formed 7x52 grid), the maximum intensity wins.
    """
    cells = grid.nonzero_cells()
    if not cells:
        return PaintPlan(day_commits=[])

    date_intensity: dict[date, int] = {}
    for row, col, intensity in cells:
        d = grid.cell_to_date(row, col)
        date_intensity[d] = max(date_intensity.get(d, 0), intensity)

    day_commits: List[Tuple[date, int]] = []
    for d, intensity in sorted(date_intensity.items()):
        count = calibration.commits_for_intensity(intensity)
        if count > 0:
            day_commits.append((d, count))

    return PaintPlan(day_commits=day_commits)
