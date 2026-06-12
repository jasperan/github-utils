"""Execution engine -- wires paint plans and simulation plans to git operations.

Takes a ``PaintPlan`` (from the painter orchestrator) or a list of ``DayPlan``
objects (from the profile engine) and creates backdated commits via
``GitEngine``, optionally applying temporal and content realism.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date, datetime, timezone
from pathlib import Path
from typing import Callable, Dict, Iterator, List, Optional, Tuple

from git import Repo

from github_utils.git.operations import GitEngine
from github_utils.painter.engine import PaintPlan
from github_utils.realism.content import ContentEngine
from github_utils.realism.profile import DayPlan
from github_utils.realism.temporal import TemporalEngine


@dataclass
class ExecutionResult:
    """Summary returned after executing a plan."""

    total_commits: int
    total_pushes: int = 0


def execute_paint_plan(
    repo_path: Path,
    plan: PaintPlan,
    use_realism: bool = True,
    push: bool = False,
    push_every: int = 500,
    seed: Optional[int] = None,
) -> ExecutionResult:
    """Execute a paint plan by creating backdated commits.

    Parameters
    ----------
    repo_path:
        Path to a local git repository.
    plan:
        A ``PaintPlan`` mapping dates to commit counts.
    use_realism:
        When ``True``, timestamps and commit content are generated via the
        temporal and content realism engines.  When ``False``, commits use
        fixed noon timestamps and minimal placeholder content.
    push:
        Push to the remote after execution.
    push_every:
        When *push* is enabled, push periodically after this many commits
        (useful for very large plans to avoid losing progress).
    seed:
        Deterministic seed passed to the realism engines.

    Returns
    -------
    ExecutionResult
        Summary with total commits created and number of push operations.
    """
    repo = Repo(repo_path)
    engine = GitEngine(repo)
    counter = 0
    pushes = 0

    # Choose a per-commit builder once, up front.  Both strategies yield
    # ``(timestamp, message, file_changes)`` for each commit of a given day;
    # ``counter`` is the running commit index (used by the fixed-noon path).
    if use_realism:
        build_day = _realism_day_builder(seed)
    else:
        build_day = _fixed_noon_day_builder

    def maybe_push() -> None:
        nonlocal pushes
        if push and push_every > 0 and counter % push_every == 0:
            engine.push()
            pushes += 1

    for day_date, count in plan.day_commits:
        for ts, msg, changes in build_day(day_date, count, counter):
            engine.create_backdated_commit(date=ts, message=msg, file_changes=changes)
            counter += 1
            maybe_push()

    if push and counter > 0 and (push_every <= 0 or counter % push_every != 0):
        engine.push()
        pushes += 1

    return ExecutionResult(total_commits=counter, total_pushes=pushes)


_DayBuilder = Callable[[date, int, int], Iterator[Tuple[datetime, str, Dict[str, str]]]]


def _realism_day_builder(seed: Optional[int]) -> _DayBuilder:
    """Return a day builder that uses temporal and content realism engines."""
    temporal = TemporalEngine(seed=seed)
    content = ContentEngine(seed=seed)

    def build(day_date: date, count: int, _counter: int) -> Iterator[Tuple[datetime, str, Dict[str, str]]]:
        for ts in temporal.generate_timestamps(day_date, count):
            # Temporal engine returns naive datetimes; attach UTC so
            # GitPython's parse_date accepts them.
            if ts.tzinfo is None:
                ts = ts.replace(tzinfo=timezone.utc)
            yield ts, content.generate_message(), content.generate_file_changes()

    return build


def _fixed_noon_day_builder(
    day_date: date, count: int, counter: int
) -> Iterator[Tuple[datetime, str, Dict[str, str]]]:
    """Yield fixed-noon commits with minimal placeholder content."""
    for i in range(count):
        extra_min, sec = divmod(i, 60)
        ts = datetime(
            day_date.year, day_date.month, day_date.day,
            12, extra_min % 60, sec, tzinfo=timezone.utc,
        )
        idx = counter + i
        yield ts, f"update {idx}", {"a.txt": str(idx)}


def execute_simulation(
    repo_path: Path,
    day_plans: List[DayPlan],
    use_realism: bool = True,
    push: bool = False,
    push_every: int = 500,
    seed: Optional[int] = None,
) -> ExecutionResult:
    """Execute a simulation by converting ``DayPlan`` objects to a ``PaintPlan``.

    Days with zero commits are skipped.
    """
    day_commits = [
        (dp.date, dp.commit_count)
        for dp in day_plans
        if dp.commit_count > 0
    ]
    plan = PaintPlan(day_commits=day_commits)
    return execute_paint_plan(
        repo_path=repo_path,
        plan=plan,
        use_realism=use_realism,
        push=push,
        push_every=push_every,
        seed=seed,
    )
