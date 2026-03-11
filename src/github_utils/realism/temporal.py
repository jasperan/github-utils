"""Temporal Realism Engine -- generates realistic timestamps for a given day.

Uses weighted hour distribution to produce timestamps that mimic real
developer activity: heavier during business hours (10am-12pm, 2pm-5pm),
lighter during early morning and late night.
"""

from __future__ import annotations

import random
from datetime import date, datetime, timedelta, timezone
from typing import List, Optional

# Default hour weights (index = hour 0..23).
# Peaks at 10-12 and 14-17 to mirror typical developer activity.
DEFAULT_HOUR_WEIGHTS: List[float] = [
    1,   # 00
    1,   # 01
    1,   # 02
    1,   # 03
    1,   # 04
    2,   # 05
    3,   # 06
    5,   # 07
    8,   # 08
    12,  # 09
    18,  # 10
    20,  # 11
    15,  # 12
    10,  # 13
    18,  # 14
    20,  # 15
    18,  # 16
    15,  # 17
    10,  # 18
    6,   # 19
    4,   # 20
    3,   # 21
    2,   # 22
    1,   # 23
]

_JITTER_MINUTES = 15  # +-15 minutes jitter around each chosen hour


class TemporalEngine:
    """Generate realistic timestamps for a target calendar day.

    Parameters
    ----------
    timezone_name:
        IANA timezone string. Currently only ``"UTC"`` is fully supported;
        the value is stored for future timezone-aware expansion.
    seed:
        Optional integer seed for deterministic output.
    hour_weights:
        Optional list of 24 floats controlling relative probability of each
        hour (index 0 = midnight).  Defaults to ``DEFAULT_HOUR_WEIGHTS``.
    """

    def __init__(
        self,
        timezone_name: str = "UTC",
        seed: Optional[int] = None,
        hour_weights: Optional[List[float]] = None,
    ) -> None:
        self.timezone_name = timezone_name
        self._rng = random.Random(seed)
        self._weights = hour_weights or list(DEFAULT_HOUR_WEIGHTS)
        if len(self._weights) != 24:
            raise ValueError("hour_weights must contain exactly 24 entries")

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def generate_timestamps(
        self,
        target_date: date,
        count: int,
    ) -> List[datetime]:
        """Return *count* sorted, realistic timestamps on *target_date*.

        Each timestamp is constructed by:
        1. Picking an hour via weighted random selection.
        2. Adding uniform-random minutes (0-59).
        3. Adding uniform-random seconds (0-59).
        4. Applying a jitter of +/-``_JITTER_MINUTES`` minutes.
        5. Clamping the result so it stays within *target_date*.
        """
        day_start = datetime(
            target_date.year,
            target_date.month,
            target_date.day,
            0, 0, 0,
        )
        day_end = datetime(
            target_date.year,
            target_date.month,
            target_date.day,
            23, 59, 59,
        )

        hours = self._rng.choices(
            population=list(range(24)),
            weights=self._weights,
            k=count,
        )

        timestamps: List[datetime] = []
        for hour in hours:
            minute = self._rng.randint(0, 59)
            second = self._rng.randint(0, 59)
            base = day_start.replace(hour=hour, minute=minute, second=second)

            # Apply jitter
            jitter = self._rng.uniform(-_JITTER_MINUTES, _JITTER_MINUTES)
            ts = base + timedelta(minutes=jitter)

            # Clamp to target date boundaries
            if ts < day_start:
                ts = day_start
            elif ts > day_end:
                ts = day_end

            timestamps.append(ts)

        timestamps.sort()
        return timestamps
