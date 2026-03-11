"""Profile Realism Engine -- models developer behavior over weeks and months.

Generates day-by-day plans with realistic commit counts based on configurable
developer profiles.  A state machine transitions between normal, burst, and
quiet modes to produce organic-looking activity patterns.  Five named profiles
ship by default: ``default``, ``night-owl``, ``weekend-warrior``, ``9-to-5``,
and ``open-source-maintainer``.
"""

from __future__ import annotations

import random
from dataclasses import dataclass, field
from datetime import date, timedelta
from enum import Enum, auto
from typing import Dict, List, Optional, Tuple


# ---------------------------------------------------------------------------
# Data structures
# ---------------------------------------------------------------------------


@dataclass
class DayPlan:
    """A single day's commit plan."""

    date: date
    commit_count: int


@dataclass
class ProfileConfig:
    """Knobs that define a developer's activity fingerprint.

    Attributes
    ----------
    weekly_pattern:
        Seven floats (Mon=0 .. Sun=6) that multiply the base commit count
        for each day of the week.
    commits_per_active_day:
        ``(min, max)`` range for commits on an active day (before multipliers).
    burst_probability:
        Probability per day of entering burst mode from normal mode.
    burst_duration_days:
        ``(min, max)`` range for how many days a burst lasts.
    burst_multiplier:
        Factor applied to commit counts during burst mode.
    quiet_probability:
        Probability per day of entering quiet mode from normal mode.
    quiet_duration_days:
        ``(min, max)`` range for how many days a quiet period lasts.
    """

    weekly_pattern: List[float] = field(
        default_factory=lambda: [1.0, 1.0, 1.0, 1.0, 1.0, 0.4, 0.3],
    )
    commits_per_active_day: Tuple[int, int] = (1, 6)
    burst_probability: float = 0.05
    burst_duration_days: Tuple[int, int] = (2, 7)
    burst_multiplier: float = 2.0
    quiet_probability: float = 0.08
    quiet_duration_days: Tuple[int, int] = (2, 5)


# ---------------------------------------------------------------------------
# State machine
# ---------------------------------------------------------------------------


class _Mode(Enum):
    NORMAL = auto()
    BURST = auto()
    QUIET = auto()


# ---------------------------------------------------------------------------
# Named profile presets
# ---------------------------------------------------------------------------

_NAMED_PROFILES: Dict[str, ProfileConfig] = {
    "default": ProfileConfig(),
    "night-owl": ProfileConfig(
        weekly_pattern=[1.2, 1.2, 1.2, 1.2, 1.0, 0.6, 0.5],
        commits_per_active_day=(1, 7),
        burst_probability=0.06,
        burst_duration_days=(2, 5),
        burst_multiplier=2.2,
        quiet_probability=0.07,
        quiet_duration_days=(1, 4),
    ),
    "weekend-warrior": ProfileConfig(
        weekly_pattern=[0.3, 0.3, 0.3, 0.3, 0.4, 1.5, 1.5],
        commits_per_active_day=(2, 8),
        burst_probability=0.08,
        burst_duration_days=(1, 3),
        burst_multiplier=2.0,
        quiet_probability=0.05,
        quiet_duration_days=(1, 3),
    ),
    "9-to-5": ProfileConfig(
        weekly_pattern=[1.0, 1.0, 1.0, 1.0, 0.8, 0.1, 0.1],
        commits_per_active_day=(2, 5),
        burst_probability=0.03,
        burst_duration_days=(2, 4),
        burst_multiplier=1.8,
        quiet_probability=0.10,
        quiet_duration_days=(2, 6),
    ),
    "open-source-maintainer": ProfileConfig(
        weekly_pattern=[1.0, 1.0, 1.0, 1.0, 1.0, 0.8, 0.7],
        commits_per_active_day=(2, 8),
        burst_probability=0.07,
        burst_duration_days=(3, 10),
        burst_multiplier=2.5,
        quiet_probability=0.04,
        quiet_duration_days=(1, 3),
    ),
}


# ---------------------------------------------------------------------------
# Engine
# ---------------------------------------------------------------------------


class ProfileEngine:
    """Generate day-by-day commit plans that mimic real developer activity.

    The engine runs a three-state machine (normal / burst / quiet) over the
    requested date range.  Each day's raw commit count is drawn uniformly from
    ``commits_per_active_day``, then multiplied by the weekday factor from
    ``weekly_pattern`` and optionally by ``burst_multiplier``.  In quiet mode
    the count is forced to zero.

    Parameters
    ----------
    config:
        A ``ProfileConfig`` instance.  Defaults to the ``"default"`` profile.
    seed:
        Optional integer seed for deterministic output.
    """

    def __init__(
        self,
        config: Optional[ProfileConfig] = None,
        seed: Optional[int] = None,
    ) -> None:
        self._config = config or ProfileConfig()
        self._rng = random.Random(seed)

    # ------------------------------------------------------------------
    # Factory
    # ------------------------------------------------------------------

    @classmethod
    def from_name(cls, name: str, seed: Optional[int] = None) -> ProfileEngine:
        """Create an engine from a named profile preset.

        Available names: ``default``, ``night-owl``, ``weekend-warrior``,
        ``9-to-5``, ``open-source-maintainer``.

        Raises
        ------
        ValueError
            If *name* is not a recognised profile.
        """
        if name not in _NAMED_PROFILES:
            available = ", ".join(sorted(_NAMED_PROFILES))
            raise ValueError(
                f"Unknown profile {name!r}. Available: {available}"
            )
        return cls(config=_NAMED_PROFILES[name], seed=seed)

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def generate_plan(
        self,
        start_date: date,
        end_date: date,
    ) -> List[DayPlan]:
        """Return a ``DayPlan`` for every day in ``[start_date, end_date]``.

        The range is inclusive on both ends.
        """
        cfg = self._config
        mode = _Mode.NORMAL
        remaining_in_mode = 0

        plans: List[DayPlan] = []
        current = start_date
        while current <= end_date:
            # --- state transitions (enter new mode when idle) ---
            if remaining_in_mode == 0 and mode == _Mode.NORMAL:
                roll = self._rng.random()
                if roll < cfg.burst_probability:
                    mode = _Mode.BURST
                    remaining_in_mode = self._rng.randint(
                        *cfg.burst_duration_days,
                    )
                elif roll < cfg.burst_probability + cfg.quiet_probability:
                    mode = _Mode.QUIET
                    remaining_in_mode = self._rng.randint(
                        *cfg.quiet_duration_days,
                    )

            # --- compute commit count for the day ---
            if mode == _Mode.QUIET:
                count = 0
            else:
                lo, hi = cfg.commits_per_active_day
                base = self._rng.randint(lo, hi)
                weekday_idx = current.weekday()  # Mon=0 .. Sun=6
                multiplier = cfg.weekly_pattern[weekday_idx]
                if mode == _Mode.BURST:
                    multiplier *= cfg.burst_multiplier
                count = max(0, round(base * multiplier))

            plans.append(DayPlan(date=current, commit_count=count))

            # --- tick down mode duration AFTER computing today ---
            if remaining_in_mode > 0:
                remaining_in_mode -= 1
                if remaining_in_mode == 0 and mode != _Mode.NORMAL:
                    mode = _Mode.NORMAL

            current += timedelta(days=1)

        return plans
