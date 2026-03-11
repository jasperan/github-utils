"""Execution engine -- wires paint plans and simulation plans to git operations.

Takes a ``PaintPlan`` (from the painter orchestrator) or a list of ``DayPlan``
objects (from the profile engine) and creates backdated commits via
``GitEngine``, optionally applying temporal and content realism.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date, datetime, timezone
from pathlib import Path
from typing import List, Optional

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
    temporal = TemporalEngine(seed=seed) if use_realism else None
    content = ContentEngine(seed=seed) if use_realism else None
    counter = 0
    pushes = 0

    for day_date, count in plan.day_commits:
        if use_realism and temporal is not None and content is not None:
            timestamps = temporal.generate_timestamps(day_date, count)
            for ts in timestamps:
                # Temporal engine returns naive datetimes; attach UTC
                # so GitPython's parse_date accepts them.
                if ts.tzinfo is None:
                    ts = ts.replace(tzinfo=timezone.utc)
                msg = content.generate_message()
                changes = content.generate_file_changes()
                engine.create_backdated_commit(
                    date=ts, message=msg, file_changes=changes,
                )
                counter += 1
                if push and push_every > 0 and counter % push_every == 0:
                    engine.push()
                    pushes += 1
        else:
            for i in range(count):
                extra_min, sec = divmod(i, 60)
                ts = datetime(
                    day_date.year, day_date.month, day_date.day,
                    12, extra_min % 60, sec, tzinfo=timezone.utc,
                )
                engine.create_backdated_commit(
                    date=ts,
                    message=f"update {counter}",
                    file_changes={"a.txt": str(counter)},
                )
                counter += 1
                if push and push_every > 0 and counter % push_every == 0:
                    engine.push()
                    pushes += 1

    if push and counter > 0 and (push_every <= 0 or counter % push_every != 0):
        engine.push()
        pushes += 1

    return ExecutionResult(total_commits=counter, total_pushes=pushes)


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
